# Backup Scope

## Backup Units

A Cloudogu backup contains several backup units that can be specified or excluded in the Backup custom resource.
Each unit covers the resources listed below.

### Global Config

The backup includes only the ConfigMap labeled `k8s.cloudogu.com/type: global-config`.

### V2 Dogus

All V2 Dogus are backed up together as a single unit.
The operator creates this unit only if the cluster contains V2 Dogus.

For each V2 Dogu, the backup includes ConfigMaps, Secrets, PersistentVolumeClaims and the Dogu resource itself
if the value of either `dogu.name` or `k8s.cloudogu.com/dogu.name` matches the Dogu name.
A ValidatingAdmissionWebhook enforces explicit labeling on the Dogu resource.
The Dogu operator sets the labels automatically on all other resources.

### Dogus

This currently refers to V3 Dogus, which are backed up individually and can therefore be excluded individually.

For each selected Dogu, the backup includes ConfigMaps, Secrets, PersistentVolumeClaims and the Dogu resource itself
if the value of `k8s.cloudogu.com/dogu.name` matches the Dogu name.

### Components

Components are backed up together as a single unit.
The operator creates this unit only if the cluster contains components with resources to back up.

The backup includes ConfigMaps, Secrets, PersistentVolumeClaims and ServiceAccountProducers that have the
`k8s.cloudogu.com/backup-scope` label and whose `k8s.cloudogu.com/component.name` label value matches the name of a
component that exists in the cluster and passes the backup's exclusion rules.

#### Components as service account producers

The component-operator adds the label `k8s.cloudogu.com/backup-scope` to every `ServiceAccountProducer` of a component.
The cleanup before a restore then deletes the producer, and the service-account-operator deletes the secrets of all
`ServiceAccountRequests` for this producer.
Once the restore recreates the producer, the service-account-operator rotates the account and recreates the secrets with
new credentials, so they match the new data.

## Restore Behavior For Component Resources

During restore, the operator deletes all resources that belong to the restore scope before recreating them from the backup.
This applies both to Dogus and to Component resources selected via `k8s.cloudogu.com/backup-scope`.
For a restore from backup units, only the restored Dogus and components belong to the restore scope:
Component resources are only deleted if their label `k8s.cloudogu.com/component.name` names a restored component.
Dogus and components excluded via `excludedDogus`, `excludedComponents` or `options.ignoreV2Dogus` are left untouched.
For a restore from a legacy backup, all Dogus and all resources with `k8s.cloudogu.com/backup-scope` are still deleted.

If a workload mounts or otherwise uses one of those Component resources, it must be scaled down before the restore starts and scaled up again after the restore finishes. 
To participate in this behavior, the workload must be labeled with `k8s.cloudogu.com/restore-scaledown-scope`.
Otherwise, Pods may continue to access resources that are being deleted and recreated.

The restore flow is:

1. Switch the system into maintenance mode.
2. Scale down workloads marked for restore scaledown.
3. Delete Dogus (including their PVCs) and Component resources that are part of the restore scope.
4. Trigger the restore at the configured provider.
5. Scale the previously scaled down workloads back up.

## Labels

The following labels are used together:

### `k8s.cloudogu.com/backup-scope`

Use this label on Component resources that should be part of the backup and restore scope.

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
