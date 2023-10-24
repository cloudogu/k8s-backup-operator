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