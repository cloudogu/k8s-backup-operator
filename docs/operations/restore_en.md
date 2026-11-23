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

## Keep the target instance's global configuration

When restoring a backup from another Dogu v2 instance, set `spec.options.ignoreGlobalConfig: true`
to keep the target instance's global configuration. The operator preserves the existing global ConfigMap during cleanup,
even if it has the `k8s.cloudogu.com/backup-scope` label. Velero skips the global ConfigMap in the backup even when the
target instance has no global ConfigMap.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Restore
metadata:
  name: restore-preserve-global-config
spec:
  provider: velero
  backupName: backup-sample
  options:
    ignoreGlobalConfig: true
```

When the option is omitted or set to `false`, the restore uses the global configuration from the backup as usual.
The option applies to each restore individually. The global configuration remains part of the backup.

## Starting a backup after a restore

A backup created during this restore waits and starts only after the restore operation has finished. This does not
guarantee that the entire Cloudogu EcoSystem and all PVCs are ready, however. A backup started immediately afterwards
may therefore partially fail. See
[Starting a backup immediately after a restore](./backup_en.md#starting-a-backup-immediately-after-a-restore) for the
recommended procedure and the rationale for this behavior.
