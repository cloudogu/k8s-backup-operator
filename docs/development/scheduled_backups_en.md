# Scheduled Backups

Scheduled backups are created by creating a [`BackupSchedule` resource](../operations/scheduled_backups_en.md).
The Backup Operator turns each created `BackupSchedule` into a Kubernetes `CronJob` named
`backup-schedule-<backup-schedule-name>` in the same namespace.

## Reconciliation

The controller watches both `BackupSchedule` resources and their owned `CronJob` resources.
During reconciliation, it:

1. adds labels and a finalizer to the `BackupSchedule`;
2. validates that `spec.schedule` contains a standard cron expression;
3. if it does, creates or updates the owned `CronJob` so that its schedule, labels, owner reference, pod template,
   operator image, and image pull secrets match the desired state; and
4. records the result in the `Accepted` and `Ready` status conditions of the `BackupSchedule`.

Changes to or deletion of the owned `CronJob` trigger another reconciliation, so the controller restores
the desired state.

## Creating a backup

At every scheduled time, the `CronJob` starts the Backup Operator image with the `scheduled-backup`
subcommand. The process creates a [`Backup` resource](../operations/backup_en.md) in the same namespace
and passes through the provider configured in the `BackupSchedule`.
The backup name consists of the `BackupSchedule` name and its creation timestamp, for example
`daily-2026-08-19t02.00.00`.

## Deletion

When a `BackupSchedule` is deleted, the controller requests deletion of the managed `CronJob` and then removes
the finalizer from the `BackupSchedule`. A missing `CronJob` is treated as already deleted and is not processed
further. If deleting the `CronJob` or removing the finalizer fails, the finalizer remains and reconciliation is
retried.

## Kubernetes events

The controller records the following events on the `BackupSchedule`:

| Type | Reason | Meaning |
|------|--------|---------|
| Normal | `CronJobCreated` | The managed `CronJob` was created. |
| Normal | `CronJobUpdated` | The managed `CronJob` was updated. |
| Normal | `CronJobDeletionRequested` | Deletion of the managed `CronJob` was requested. |
| Warning | `InvalidSchedule` | `spec.schedule` is invalid. |
| Warning | `CronJobSynchronizationFailed` | Creating or updating the managed `CronJob` failed. |
| Warning | `CronJobDeletionFailed` | Deleting the managed `CronJob` failed. |
| Warning | `FinalizerRemovalFailed` | Removing the `BackupSchedule` finalizer failed. |
