# Zeitgesteuerte Backups

Backups lassen sich automatisiert und zeitgesteuert durchführen.
Dies lässt sich mit einer `BackupSchedule`-Ressource erreichen:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: BackupSchedule
metadata:
  name: backupschedule-sample
spec:
  schedule: "0 0 * * *" # ein Cron-Pattern welches die Ausführung des Backups bestimmt.
  provider: "velero" # aktuell wird nur velero unterstützt
```

`schedule` ist ein Cron-Pattern wie es in der [Kubernetes CronJob Syntax](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#schedule-syntax) definiert ist.