# Exclude resources during a restore

The restore is divided into two steps. Firstly, a cleanup is performed. This cleanup deletes all dogus.

In the second step, the backup provider is used to perform a restore and add all resources from the
backup to the cluster.

## Exclude resources in the restore process

A [Plugin for excluding resources from the backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/) 
exists for the restore provider `velero`. 
This plugin can be applied and used with `velero` in the cluster to exclude resources during the
restore process. For more information see [here](https://github.com/cloudogu/k8s-velero/blob/develop/docs/exclude_out_of_restore_en.md)
