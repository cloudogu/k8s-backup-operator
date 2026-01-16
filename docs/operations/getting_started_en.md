# Getting started

## Backup process
This guide contains all the steps required to set up backup in a Kubernetes cluster and perform backups and restores.
The individual steps should be performed in sequence and are described in the following sections:

1. Configure backup bucket
* [Google Cloud](./setup_gcloud_backup_bucket_en.md)
    * AWS
    * Metalstack
    * Minio
        * Requires [Longhorn](./use_longhorn_storage_provisioner_en.md)
        * For installation and configuration, see [Local test environment](../development/local_dev_setup_en.md)
2. [Install Velero](./installing_velero_en.md)
3. [Install Backup Operator](./backup_operator_installation_en.md)
4. [Create backup](./backup_en.md)
5. [Restore backup (create restore)](./restore_en.md)

## Further instructions
- [Schedule backups](./scheduled_backups_en.md)
- [Retention strategies for backups](./automated_backup_deletion_and_retention_en.md)
- [Use Longhorn as a storage provisioner](./use_longhorn_storage_provisioner_en.md)
