# Automated Backup Deletion and Retention

Configuring garbage-collection of backups and retention [is documented elsewhere](../operations/automated_backup_deletion_and_retention_en.md).

Here's how it is implemented:
The k8s-backup-operator has a `gc` subcommand starts it in garbage-collection mode.
This enables us to use the same image in a `CronJob` to delete backups regularly, according to the configured retention strategy.

While configuration should mainly happen through the component's values, the retention strategy is templated to and read from the ConfigMap `k8s-backup-operator-retention`.
