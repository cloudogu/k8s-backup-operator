# Getting started

## Architektur Backup

<img src="../resources/backup.svg" alt="Backup Architektur">

Im Diagramm sind alle Komponenten und deren Interaktionen dargestellt. Der ``k8s-backup-operator`` beobachtet die
Erstellung von ``k8s.cloudogu.com/Backup``-Ressourcen. Diese werden entweder vom Benutzer oder über einen Backup-Schedule erstellt. Bei der
Erstellung des Backups über einen Backupschedule übernimmt der ``k8s-backup-operator`` die Erstellung des ``k8s.cloudogu.com/Backup``.
Nach der Erstellung eines ``k8s.cloudogu.com/Backup`` erstellt der ``k8s-backup-operator`` ein ``velero.io/Backup``.
``Velero`` beobachtet diese Ressourcen, um den eigentlichen Backup-Prozess zu starten. Der Backup-Prozess ist zweigeteilt:

**Velero-Metadaten**

Velero erstellt Dateien mit Metadaten zum Backup, die in dem in Velero konfigurierten S3-Bucket abgelegt werden.

**Volume-Snapshots**

Die Volumesnapshots werden von dem im Cluster vorhandenen ``CSI (Container Storage Interface)`` erstellt. Je nachdem,
welches CSI benutzt wird, werden die Nutzdaten an unterschiedlichen Orten abgelegt. Google Cloud erstellt Volumesnapshots,
Longhorn legt die Daten in einem weiteren S3-Bucket ab.

Alle Daten des Backup sollten außerhalb des Clusters gespeichert werden.

## Backup-Prozess
Diese Anleitung enthält alle Schritte, um das Backup in einem Kubernetes Cluster einzurichten und Backups und Restores
auszuführen. Die einzelnen Schritte sollten nacheinander ausgeführt werden und werden in den folgenden Abschnitten beschrieben:

1. [Velero installieren](./installing_velero_de.md)
2. [Backup-Operator installieren](./backup_operator_installation_de.md)
3. [Backup erstellen](./backup_de.md)
4. [Backup wiederherstellen (Restore erstellen)](./restore_de.md)

### weiterführende Anleitungen
- [Backups planen](./scheduled_backups_de.md)
- [Aufbewahrungsstrategien für Backups](./automated_backup_deletion_and_retention_de.md)
- [Longhorn als Storage-Provisioner verwenden](./use_longhorn_storage_provisioner_de.md)

## Beispiele
* [Google Cloud](./setup_gcloud_backup_bucket_en.md)
* [Longhorn](./use_longhorn_storage_provisioner_en.md)
* For die Installation im lokalen Cluster: [Lokale Testumgebung](../development/local_dev_setup_de.md)