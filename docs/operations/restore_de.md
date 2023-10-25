# Restores ausführen

Eine Ressource analog zu diesem Beispiel muss im Namespace des `k8s-backup-operator` angelegt werden:
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Restore
metadata:
  name: restore-sample
spec:
  provider: velero # aktuell wird nur velero unterstützt
  backupName: backup-sample # der Name des Backups das eingespielt werden soll
```

Bevor die Wiederherstellung ausgeführt wird,
werden Ressourcen in diesem Namensraum, die für den Sicherungsprozess irrelevant sind, entfernt, um einen sauberen Zustand herzustellen.
Dies ist besonders notwendig
da installierte Dogus, die nicht in der Sicherung enthalten sind, nicht mehr funktionieren würden, weil die Sicherung ihre Datenbank nicht enthält.