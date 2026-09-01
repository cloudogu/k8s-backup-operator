# Backup creation

To back up the Cloudogu EcoSystem one has to apply a backup custom resource:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Backup
metadata:
  name: backup-sample
spec:
  provider: velero # only velero and "" (defaults to velero) is supported.
```

## Starting a backup immediately after a restore

Backup and restore operations cannot run at the same time. If a backup is created while a restore is running, the
backup waits until the restore operation has finished. Internally, this coordination uses a shared Kubernetes lease.

Finishing the restore operation does not guarantee that the entire Cloudogu EcoSystem has finished starting or that
every PersistentVolumeClaim (PVC) is ready and bound. A backup that starts immediately afterwards can therefore
partially fail because some PVCs are not available yet.

Before creating a backup after a restore, check that all components and PVCs that must be included in the backup are
ready. This is especially important for scheduled backups: avoid scheduling one so close to a restore that it starts
as soon as the restore operation finishes. If a backup of the currently available data is more important than waiting
for the complete EcoSystem, the backup can still be started deliberately; inspect its result for partial failures.

There is intentionally no global EcoSystem health check before a backup or at the end of a restore. Such a check would
prevent backing up a degraded or partially running EcoSystem even when preserving the available data is useful. It
could also prevent a restore from completing when a partially failed backup can restore only some components, even
though recovering that data may still be useful. The coordination between backup and restore consequently guarantees
that the operations do not run at the same time, but not that the entire EcoSystem is ready.
