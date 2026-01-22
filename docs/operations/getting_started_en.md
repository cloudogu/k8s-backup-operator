# Getting started

## Backup architecture

<img src="../resources/backup.svg" alt="Backup Architektur">

The diagram shows all components and their interactions. The ``k8s-backup-operator`` monitors the
creation of ``k8s.cloudogu.com/Backup`` resources. These are created either by the user or via a backup schedule. When
creating the backup via a backup schedule, the k8s-backup-operator takes over the creation of k8s.cloudogu.com/Backup.
After creating a ``k8s.cloudogu.com/Backup``, the ``k8s-backup-operator`` creates a ``velero.io/Backup``.
``Velero`` monitors these resources to start the actual backup process. The backup process is divided into two parts:

**Velero metadata**

Velero creates files with metadata for the backup, which are stored in the S3 bucket configured in Velero.

**Volume snapshots**

The volume snapshots are created by the ``CSI (Container Storage Interface)`` available in the cluster. Depending on
which CSI is used, the user data is stored in different locations. Google Cloud creates volume snapshots,
Longhorn stores the data in another S3 bucket.

All backup data should be stored outside the cluster.

## Backup process
This guide contains all the steps required to set up backup in a Kubernetes cluster and perform backups and restores.
The individual steps should be performed in sequence and are described in the following sections:

1. [Install Velero](./installing_velero_en.md)
2. [Install Backup Operator](./backup_operator_installation_en.md)
3. [Create backup](./backup_en.md)
4. [Restore backup (create restore)](./restore_en.md)

## Further instructions
- [Schedule backups](./scheduled_backups_en.md)
- [Retention strategies for backups](./automated_backup_deletion_and_retention_en.md)
- [Use Longhorn as a storage provisioner](./use_longhorn_storage_provisioner_en.md)

## Examples
* [Google Cloud](./setup_gcloud_backup_bucket_en.md)
* [Longhorn](./use_longhorn_storage_provisioner_en.md)
* For installation and configuration, see [Local test environment](../development/local_dev_setup_en.md)