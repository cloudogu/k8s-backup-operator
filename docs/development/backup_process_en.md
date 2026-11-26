# Backup Process

A CES backup represents a backup run. For a new run, the backup controller determines which backup units are needed
and creates a Velero backup for each unit. `ProviderSucceeded` summarizes the results across those units.
[Backup Scope](backup_scope_en.md) describes which resources belong to each unit.

The Velero synchronization controller imports legacy Velero backups as CES backup resources with the same names.
[Backup Synchronization](backup_sync_en.md) explains how the controller synchronizes backups after a restore.

## Local Backup Workflow

The controller first checks the prerequisites. It then acquires the namespace's
[shared lease for backup and restore](parallel_access_en.md) and activates maintenance mode. It creates any missing
Velero backups and checks their progress in subsequent reconciliations. Status-only changes do not trigger another
reconciliation in the backup controller. The controller therefore schedules another reconciliation while provider work
is pending.

`ProviderSucceeded` reports the Velero outcome. `Succeeded` records the completion of the CES backup run. Before
marking the run complete, the controller deactivates maintenance mode and then releases the lease. The lease holder
owns maintenance mode. If deactivation fails, the run stays in the finalization phase.

## Cancellation and Deletion

If the time window (`retryTimeLimit`) expires, the CES backup run can end while Velero is still working. The controller
deactivates maintenance mode and releases the lease so the canceled run stops blocking the system. The CES backup
custom resource (CR) is retained as a record of the failure. Velero backups that were still running at the time of
cancellation are deleted. This cleanup is intended to prevent restores from using data that Velero may have captured
after maintenance mode ended.

A deletion request for the CES backup CR takes precedence over the normal workflow. The associated Velero backups are
deleted.

## Code Entry Points

- The [backup controller](../../internal/controller/backup/controller.go) selects the reconciliation path and manages
  retries.
- The [Velero backup creation logic](../../internal/controller/backup/unit_backups.go) defines how each unit is backed up.
  The [unit completion logic](../../internal/controller/backup/unit_completion.go) aggregates the results.
- The [reconciler](../../internal/controller/backup/reconciler.go) checks prerequisites, manages maintenance mode,
  and handles cancellation and deletion.
- The [Velero synchronization controller](../../internal/controller/backup/velero_backup_controller.go) determines
  which Velero backups to synchronize.
