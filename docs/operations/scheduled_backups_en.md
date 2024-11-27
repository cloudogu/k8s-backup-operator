# Scheduled backups

Backups can be automated and scheduled.
This can be achieved with a `BackupSchedule` resource:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: BackupSchedule
metadata:
  name: backupschedule-sample
spec:
  schedule: "0 0 * *" # the cron pattern according to which the backups should be executed.
  provider: "velero" # only velero and "" (velero by default) are supported.
```

`schedule` is a cron pattern as defined in [Kubernetes CronJob Syntax](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#schedule-syntax).