# Geplante Backups

Geplante Backups werden [durch die Anwendung einer `BackupSchedule`-Ressource](../operations/scheduled_backups_en.md) erstellt.
Der Backup-Operator erstellt daraufhin einen `CronJob`, der [`Backup`-Ressourcen](../operations/backup_en.md) gemäß dem angegebenen Zeitplan erstellt.

Dieser `CronJob` nutzt einen `kubectl`-Pod,
der wiederum eine `ConfigMap` einbindet, die ein Shell-Skript zum Erstellen der `Backup`-Ressource enthält.
Der Name dieser `Backup`-Ressource enthält den Zeitstempel ihrer Erstellung.