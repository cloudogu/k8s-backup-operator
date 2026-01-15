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