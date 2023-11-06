# Scheduled Backups

Scheduled backups are created [by applying a `BackupSchedule` resource](../operations/scheduled_backups_en.md).
The Backup Operator then creates a `CronJob` that creates [`Backup` resources](../operations/backup_en.md) with the given schedule.

This `CronJob` makes use of a `kubectl` pod
which in turn mounts a `ConfigMap` containing a shell script for creating the `Backup` resource.
The name of this `Backup` resource contains the timestamp of its creation.
