# Getting started

## Backup-Prozess
Diese Anleitung enthält alle Schritte, um das Backup in einem Kubernetes Cluster einzurichten und Backups und Restores
auszuführen. Die einzelnen Schritte sollten nacheinander ausgeführt werden und werden in den folgenden Abschnitten beschrieben:

1. Backup-Bucket konfigurieren
    * [Google Cloud](./setup_gcloud_backup_bucket_de.md)
    * AWS
    * Metalstack
    * Minio
      * benötigt [Longhorn](./use_longhorn_storage_provisioner_de.md)
      * Installation und Konfiguration siehe [Lokale Testumgebung](../development/local_dev_setup_de.md)
2. [Velero installieren](./installing_velero_de.md)
3. [Backup-Operator installieren](./backup_operator_installation_de.md)
4. [Backup erstellen](./backup_de.md)
5. [Backup wiederherstellen (Restore erstellen)](./restore_de.md)

## weiterführende Anleitungen
- [Backups planen](./scheduled_backups_de.md)
- [Aufbewahrungsstrategien für Backups](./automated_backup_deletion_and_retention_de.md)
- [Longhorn als Storage-Provisioner verwenden](./use_longhorn_storage_provisioner_de.md)