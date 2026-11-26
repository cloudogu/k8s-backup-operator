# Restore-Prozess

Ein CES-Restore stellt ein CES-Backup im Namespace wieder her. Der Restore-Controller ermittelt, welche Backup-Units des
Backups wiederhergestellt werden, und erstellt für jede Unit ein entsprechendes Velero-Restore.
`ProviderSucceeded` fasst die Ergebnisse der Units zusammen. Mit `spec.excludedDogus`, `spec.excludedComponents` und den 
Optionen des Restores können nur ausgewählte Teile des Backups wiederhergestellt werden. Ein älteres Backup ohne Units 
wird als Ganzes durch ein einzelnes Velero-Restore mit dem gleichen Namen des CES-Restores wiederhergestellt.

Der Controller arbeitet in Stages: Jeder Reconcile führt die Stages der Reihe nach aus, bis eine davon das Restore
schreibt, warten muss oder fehlschlägt. Conditions halten den Fortschritt fest, führend bleiben aber immer die Ressourcen 
im Cluster. Ein neu gestarteter Operator setzt deshalb dort fort, wo der vorherige Lauf aufgehört hat. Anders als der
Backup-Controller beobachtet der Restore-Controller auch seine eigenen Velero-Restores. Deren Phasenwechsel lösen den
nächsten Reconcile direkt aus. Zeitgesteuerte Requeues dienen ab Erstellung der Velero-Ressourcen in der Regel nur noch 
als Sicherheitsnetz.

## Ablauf eines Restores

1. Der Controller initialisiert die Conditions `Succeeded`, `Prepared`, `ProviderSucceeded` und `WorkloadsRecovered`
   mit `Unknown`, setzt Finalizer und Labels und übernimmt das
   [gemeinsame Backup-/Restore-Lease](parallel_access_de.md).
2. Bevor etwas Destruktives passiert, prüft er, ob Velero bereit ist und ob sich das Backup wie angefordert
   wiederherstellen lässt: Das Backup muss existieren, die Ausschlüsse müssen dazu passen und die
   wiederherzustellenden Backup-Units müssen verwendbar sein. Außerdem darf kein Velero-Restore mit einem der erwarteten
   Namen unter einem anderen Owner im Cluster liegen. Schlägt eine Prüfung fehl, endet der Restore mit `Succeeded=False`.
3. Er aktiviert den Wartungsmodus (Best Effort), skaliert die Workloads herunter und löscht die Ressourcen, die
   wiederhergestellt werden (`Prepared=True`).
4. Er erstellt die Velero-Restores der Units und wartet auf ihre Ergebnisse (`ProviderSucceeded`).
5. Er skaliert die Workloads wieder hoch, wartet bis sie bereit sind, entfernt die temporären Replica-Labels und
   deaktiviert den Wartungsmodus. Erst dann setzt er `WorkloadsRecovered=True` und `Succeeded=True`, womit das Lease
   frei wird.

Die Vorbereitung läuft nur einmal: Ist `Prepared=True` gesetzt oder existiert ein Velero-Restore, mit diesem Restore
als Owner, werden Vorbereitung und Prüfungen übersprungen.

## Fehler und Löschung

Schlägt der Velero-Restore einer Unit fehl, setzt der Controller `Succeeded=False` und deaktiviert den Wartungsmodus.
Die Workloads bleiben bewusst herunterskaliert, weil die Daten, mit denen sie starten würden, unvollständig sind. Da
`Succeeded=False` terminal ist, wird das Lease frei und ein anderes Backup oder ein anderer Restore kann starten. Fehler
in anderen Schritten werden mit Backoff wiederholt.

Eine Löschung hat vor dem normalen Ablauf Vorrang. Der Controller löscht seine eigenen Velero-Restores und
wartet, bis sie tatsächlich entfernt sind, gibt das Lease frei und entfernt dann seinen Finalizer. Velero-Restores, die
jemand anderem gehören, bleiben erhalten und blockieren die Löschung nicht.

Restores aus älteren Operator-Versionen haben nur das veraltete Feld `status.status`. Der Controller übernimmt es einmalig
in eine `Succeeded`-Condition und schreibt `status.status` für ältere Clients weiterhin mit.

## Einstieg im Code

- Der [Restore-Reconciler](../../internal/controller/restore/restoreReconciler.go) legt die Reihenfolge der Stages für
  Erstellung und Löschung fest; [stageOutcome.go](../../internal/controller/restore/stageOutcome.go) legt fest, wie eine
  Stage fortsetzt, wartet oder fehlschlägt.
- Der [Restore-Zustand](../../internal/controller/restore/state) sammelt die Backup-Units des Quell-Backups und
  entscheidet, welche Units wiederhergestellt werden.
- Die [Velero-Restore-Logik](../../internal/provider/velero/restore.go) erstellt, beobachtet und löscht die
  Velero-Restores.
- [scale.go](../../internal/controller/restore/scale.go) skaliert die Workloads herunter und wieder hoch. 
  [cleanup.go](../../internal/cleanup/cleanup.go) übernimmt das Aufräumen der wiederherzustellenden Ressourcen.

## Akzeptanztests

Die Restore-Specs in [`specs`](../../specs) sind destruktiv: Sie räumen den Namespace `ecosystem` auf. Sie dürfen nur
gegen einen Wegwerf-Cluster laufen, wobei `K8S_TEST_CLUSTER_KUBECONFIG` auf genau eine Kubeconfig-Datei zeigt:

```bash
K8S_TEST_CLUSTER_KUBECONFIG=/absolute/path/to/kubeconfig make acceptance-test-restore
```

`GINKGO_FOCUS='<regex>'` führt eine einzelne Spec aus.
