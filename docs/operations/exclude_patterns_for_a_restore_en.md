# Exclude resources during a restore

The restore is divided into two steps. Firstly, a cleanup is performed. This cleanup deletes all
resources of the cluster that are not required for the backup. All resources required by the backup stack,
have the annotation `k8s.cloudogu.com/part-of: backup`.

In the second step, the backup provider is used to perform a restore and add all resources from the
backup to the cluster.

## Exclude ressources in the cleanup

The `cleanup.exclude` attribute in `values.yaml` can be used to exclude any resources from the cleanup.
The resources only need to be specified in the GVKN pattern (group, version, kind, name). 
```yaml
...
cleanup:
  exclude:
    - name: "k8s-backup-operator"
      kind: "Component"
      version: "*"
      group: "k8s.cloudogu.com"
    - name: "test-certificate"
      kind: "Secret"
      version: "*"
...
```
By default, all resources that are required for the backup, the ces load balancer and the certificate are excluded from the cleanup. 
These resources are retained after the cleanup.

## Exclude resources in the restore process

A [Plugin for excluding resources from the backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/) 
exists for the restore provider `velero`. 
This plugin can be applied and used with `velero` in the cluster to exclude resources during the
restore process. For more information see [here](https://github.com/cloudogu/k8s-velero/blob/develop/docs/exclude_out_of_restore_en.md)

## Interaction

These two exclusion processes should be used together. This makes it possible to keep resources
in the cluster before importing a restore and to continue using them afterwards.