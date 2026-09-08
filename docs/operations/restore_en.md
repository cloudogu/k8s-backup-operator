# Execute restores

A resource analogous to this example must be created in the namespace of the `k8s-backup-operator`:
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Restore
metadata:
  name: restore-sample
spec:
  provider: velero # currently only velero is supported
  backupName: backup-sample # the name of the backup you want to restore
```

Before the restore is executed,
resources in this namespace which are irrelevant to the backup process are removed to provide a clean slate.
This is especially necessary
as installed dogus not included in the backup would be broken because the backup does not contain their database.

## Starting a backup after a restore

A backup created during this restore waits and starts only after the restore operation has finished. This does not
guarantee that the entire Cloudogu EcoSystem and all PVCs are ready, however. A backup started immediately afterwards
may therefore partially fail. See
[Starting a backup immediately after a restore](./backup_en.md#starting-a-backup-immediately-after-a-restore) for the
recommended procedure and the rationale for this behavior.
