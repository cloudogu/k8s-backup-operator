# Exclude files during a restore

The restore is divided into two steps. Firstly, a cleanup is performed. This cleanup deletes all
resources of the cluster that are not required for the backup. All resources required by the backup stack,
have the annotation `k8s.cloudogu.com/part-of: backup`.

In the second step, the backup provider is used to perform a restore and add all resources from the
backup to the cluster.

## Exclude files in the cleanup

The `cleanup.exclude` attribute in `values.yaml` can be used to exclude any resources from the cleanup.
The resources only need to be specified in the GVKN pattern (group, version, kind, name). By default,
all resources that are required for the backup, the ces load balancer and the certificate are excluded from the cleanup. 
These resources are retained after the cleanup.

## Exclude files in the restore process

A [Plugin for excluding resources from the backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/) 
exists for the restore provider `velero`, which is installed. This plugin can be used with `velero` in the cluster and 
configured with the attribute `excludedFromRestores`. The configuration options are the same as for
Exclude from cleanup via the GVKN pattern. Resources that are excluded here and are present in the backup that is to be imported are ignored during this restore.

## Interaction

These two exclusion processes should be used together. This makes it possible to keep resources that are
in the cluster before importing a restore and to use them afterwards.