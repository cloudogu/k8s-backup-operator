# Automatisches Löschen und Aufbewahren von Backups

Die Konfiguration der Garbage Collection für Backups und der Aufbewahrungsdauer [ist an anderer Stelle dokumentiert](../operations/automated_backup_deletion_and_retention_en.md).

So wird dies umgesetzt:
Der k8s-backup-operator verfügt über den Unterbefehl `gc`, der ihn im Garbage-Collection-Modus startet.
Dadurch können wir dasselbe Image in einem `CronJob` verwenden, um Backups regelmäßig gemäß der konfigurierten Aufbewahrungsstrategie zu löschen.

Während die Konfiguration hauptsächlich über die Values der Komponente erfolgen sollte, wird die Aufbewahrungsstrategie anhand der ConfigMap `k8s-backup-operator-retention` bestimmt und aus dieser ausgelesen.