# Gemeinsames Lease für Backup und Restore

Backup und Restore verwenden pro Namespace dasselbe Kubernetes-`Lease` `k8s-backup-operator-lease`. Dadurch kann immer nur eine der beiden Operationen ihren kritischen Abschnitt ausführen. Die generische Acquire-, Resolver- und Release-Logik liegt in `internal/leases`; die Controller binden sie als eigene Workflow-Stages ein.

## Holder-Identität

Das Lease identifiziert seinen Holder über drei Werte:

| Feld | Inhalt |
|---|---|
| `spec.holderIdentity` | UID der Backup- oder Restore-Ressource |
| `k8s.cloudogu.com/backup-operator-lease-holder-name` | Name der Ressource |
| `k8s.cloudogu.com/lease-holder-kind` | `Backup` oder `Restore` |

Kind, Name und UID müssen zusammenpassen, damit sich eine Ressource als Holder erkennt oder ihr eigenes Lease freigeben darf.

## Erwerb und Warten

Ein fehlendes Lease wird für die aktuelle Ressource angelegt. Ist sie bereits der Holder, darf ihr Workflow fortfahren. Andernfalls wartet sie und versucht es in einem späteren Reconcile erneut.

Jeder Controller registriert nur den Resolver für seinen eigenen Ressourcentyp. Ein Lease des jeweils anderen Typs wird daher als aktiv behandelt, bis der zuständige Controller es freigibt. Ein wartender Restore setzt `Successful=Unknown/WaitingForActiveRestore` und beginnt keine destruktive Stage; ein Backup liefert entsprechend `Retry`.

Ein Zeitablauf macht das Lease nicht ungültig und es wird während einer laufenden Operation nicht periodisch erneuert. `acquireTime`, `renewTime` und `leaseTransitions` werden nur beim Anlegen oder Übernehmen aktualisiert.

## Ungültige Leases und Übernahme

UID, Name und Kind werden immer gemeinsam geschrieben. Fehlt eines dieser Felder, gilt das Lease als ungültig und wird weder aus anderen Ressourcen rekonstruiert noch automatisch repariert. Ein Restore dokumentiert das mit `Successful=Unknown/InvalidRestoreLease`; Fehler werden mit dem Backoff von `controller-runtime` erneut geprüft.

Ein strukturell vollständiges Lease kann übernommen werden, wenn sein Holder nicht mehr existiert, seine UID nicht mehr zum benannten Objekt passt oder er terminal ist. Ein unbekannter oder vom Controller nicht auflösbarer Holder-Typ wird aus Sicherheitsgründen nicht übernommen.

Lease-Änderungen verwenden die Kubernetes-`resourceVersion` als optimistische Sperre. Nach einem Konflikt liest der nächste Reconcile den aktuellen Zustand erneut.

## Freigabe

Backups geben ihr eigenes Lease frei, sobald sie erfolgreich, fehlgeschlagen, abgebrochen oder zum Löschen markiert sind. Terminale Restores tun dies im Ignore-Workflow; beim Löschen eines Restore erfolgt die Freigabe nach der bestätigten Entfernung des Provider-Childs und vor dem Entfernen des Parent-Finalizers.

Die Freigabe prüft Kind, Name und UID. UID- und `resourceVersion`-Preconditions beim Löschen verhindern, dass ein inzwischen neu vergebenes Lease versehentlich entfernt wird.
