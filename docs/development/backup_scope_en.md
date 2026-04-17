# Backup Scope

Currently, backup of single Dogus is not yet supported.
For now, the data of all Dogus and the global config is backed up.
Specifically, the resources with the label key `dogu.name` and the `k8s.cloudogu.com/type: global-config` label are selected.
Only ConfigMaps, Secrets, PersistentVolumeClaims and the Dogu resource itself are backed up.
Because the Dogu resource is backed up, the Dogu operator can recreate any other resources that are not included in the backup.

Additional resources can be included in the backup by adding the label `k8s.cloudogu.com/backup-scope` to them, for example on component `PersistentVolumeClaims`.
Be aware that the limitation to the aforementioned kinds of resources still applies.

## Restore Behavior For Additional Resources

During restore, the operator deletes all resources that belong to the restore scope before recreating them from the backup.
This applies both to Dogus and to additional resources selected via `k8s.cloudogu.com/backup-scope`.

If a workload mounts one of those additional resources, it must be scaled down before the restore starts and scaled up again after the restore finishes. 
To participate in this behavior, the workload must be labeled with `k8s.cloudogu.com/restore-scaledown-scope`.
Otherwise, Pods may continue to access resources that are being deleted and recreated.

The restore flow is:

1. Switch the system into maintenance mode.
2. Scale down workloads marked for restore scaledown.
3. Delete Dogus and additional resources that are part of the restore scope.
4. Trigger the restore at the configured provider.
5. Scale the previously scaled down workloads back up.

## Labels

The following labels are used together:

### `k8s.cloudogu.com/backup-scope`

Use this label on additional resources that should be part of the backup and restore scope.

Example:

```yaml
metadata:
  labels:
    k8s.cloudogu.com/backup-scope: component-a
```

### `k8s.cloudogu.com/restore-scaledown-scope`

Use this label on workloads that mount or otherwise depend on resources labeled with `k8s.cloudogu.com/backup-scope`.
The operator currently only checks whether this label exists. The concrete label value is not evaluated during scale-down or scale-up.

Example:

```yaml
metadata:
  labels:
    k8s.cloudogu.com/restore-scaledown-scope: component-a
```

This means:

- resources with `k8s.cloudogu.com/backup-scope: component-a` are deleted and restored as part of that scope
- workloads with `k8s.cloudogu.com/restore-scaledown-scope` are scaled down before the restore and scaled up afterwards

In practice, the value can still be used for documentation and operational clarity, but it is not interpreted by the backup operator.

### `k8s.cloudogu.com/restore-scaledown-replicas`

This label is managed by the backup operator during restore.
When a workload is scaled down, the operator stores the original replica count in this label and uses it afterwards to restore the previous scale.

Do not set or manage this label manually.

## Example

If a component uses a PVC that should be backed up and restored:

1. Label the PVC with `k8s.cloudogu.com/backup-scope: component-a`.
2. Label every workload that mounts this PVC with `k8s.cloudogu.com/restore-scaledown-scope: component-a`.
3. Do not set `k8s.cloudogu.com/restore-scaledown-replicas` yourself; it is written and removed by the operator during restore.

The example uses the same value on both labels for readability, but the current implementation only requires the presence of `k8s.cloudogu.com/restore-scaledown-scope`.
