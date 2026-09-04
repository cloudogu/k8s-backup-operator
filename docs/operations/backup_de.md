# Erstellung eines Backups

Für ein Backup des Cloudogu EcoSystem ist die Erstellung einer Backup-Ressource notwendig:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Backup
metadata:
  name: backup-sample
spec:
  provider: velero # aktuell wird nur Velero unterstützt ("" ist ein Spezialfall und wählt Velero als Provider aus)
```

## Backup unmittelbar nach einem Restore starten

Backup und Restore können nicht gleichzeitig ausgeführt werden. Wird während eines laufenden Restores ein Backup
erstellt, wartet das Backup, bis der Restore-Vorgang abgeschlossen ist. Intern wird diese Koordination über ein
gemeinsames Kubernetes-Lease umgesetzt.

Der Abschluss des Restore-Vorgangs garantiert nicht, dass das gesamte Cloudogu EcoSystem bereits vollständig gestartet
ist oder alle PersistentVolumeClaims (PVCs) bereit und gebunden sind. Ein Backup, das unmittelbar danach startet, kann
daher teilweise fehlschlagen, weil einige PVCs noch nicht verfügbar sind.

Vor einem Backup nach einem Restore sollte geprüft werden, ob alle Komponenten und PVCs bereit sind, die im Backup
enthalten sein müssen. Das ist insbesondere bei geplanten Backups wichtig: Ein Backup sollte nicht so nah an einem
Restore geplant werden, dass es direkt nach Abschluss des Restore-Vorgangs startet. Ist die Sicherung der aktuell
verfügbaren Daten wichtiger als das Warten auf das vollständige EcoSystem, kann das Backup dennoch bewusst gestartet
werden. Das Ergebnis sollte anschließend darauf geprüft werden, ob das Backup teilweise fehlgeschlagen ist.

Vor einem Backup und am Ende eines Restores wird bewusst keine globale Zustandsprüfung des EcoSystems durchgeführt.
Eine solche Prüfung würde verhindern, dass ein beeinträchtigtes oder nur teilweise laufendes EcoSystem gesichert wird,
obwohl die Sicherung der verfügbaren Daten sinnvoll sein kann. Sie könnte außerdem verhindern, dass ein Restore
abgeschlossen wird, wenn aus einem teilweise fehlgeschlagenen Backup nur einige Komponenten wiederhergestellt werden
können, obwohl auch diese Daten wertvoll sein können. Die Koordination zwischen Backup und Restore garantiert daher,
dass die Operationen nicht gleichzeitig ausgeführt werden, nicht aber die Betriebsbereitschaft des gesamten EcoSystems.

## Timeout eines Backups

Ein Backup, das nicht innerhalb von `retryTimeLimit` (Schlüssel der ConfigMap `k8s-backup-operator-backup-config`,
Standard 60 Minuten) fertig wird, wird abgebrochen – auch dann, wenn der Backup-Provider noch läuft. Dieser lässt sich von außen
nicht stoppen, aber Wartungsmodus wird planmäßig freigegeben statt ihn unnötig aktiv zu halten. Die
Backup-Ressource bleibt als fehlgeschlagenes Backup erhalten.

Das Provider-Backup eines abgebrochenen Laufs läuft weiter und wird gelöscht, sobald es beendet ist. Der Wartungsmodus
war zu diesem Zeitpunkt bereits abgeschaltet, die Daten sind daher potenziell inkonsistent und dürfen nicht
wiederhergestellt werden.

Ein kurz nach einem Abbruch erstelltes Backup wartet, bis das aufgegebene Provider-Backup beendet ist, und kann deshalb
in sein eigenes Timeout laufen. Laufen Backups wiederholt in ein Timeout, sollte `retryTimeLimit` erhöht werden.
