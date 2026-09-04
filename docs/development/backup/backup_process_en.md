# Backup process

This documentation describes the `k8s-backup-operator` backup workflow from a developer's perspective.

The backup area consists of two controllers:

- The **backup controller** processes `k8s.cloudogu.com/v1` `Backup` resources, checks prerequisites, activates maintenance mode, creates and observes the provider backup, and handles deletions.
- The **Velero backup synchronization controller** observes Velero `Backup` resources. It creates missing CES backup CRs for backups that already exist at the provider and links deletions in both directions.

The provider currently implemented is Velero. CES backups and Velero backups use the same name and namespace.

## Controller flow

The backup controller selects an operation-specific list of `ensure...` methods based on the state of the backup CR. Each stage returns an action:

| Action | Meaning |
|---|---|
| `Next` | The next stage runs during the same reconciliation. |
| `Retry` | The reconciliation ends with the configured `requeueAfter`. |
| `Abort` | The reconciliation ends without an explicit requeue. |

An error returned at the same time is passed to `controller-runtime` and triggers its backoff. The requeue interval comes from `operatorConfig.RequeueTimeSeconds`.

Unlike the restore controller, the backup controller uses an event filter. Reconciliations are triggered when the `generation` changes and when the `deletionTimestamp` is set for the first time. Changes limited to the status of the backup CR do not trigger another event through this controller. Waiting for provider progress therefore uses `Retry` instead of watching owned children. Owning the children was never an option here: a Velero backup can already exist before its CES `Backup` CR does, so the CR cannot be its owner.

## Stage order

The `deletionTimestamp` takes precedence over all conditions. Without a deletion request, backups with a terminal `Succeeded` condition enter the ignore path; a terminal `ProviderSucceeded` condition or `Canceled=True` enters the finalize path. All other backups run through this create path:

1. `ensureVeleroStatusSynced`
2. `ensureBackupSetup`
3. `ensureBackupIsCanceledAfterTimeWindowExpired`
4. `ensureBackupIsPrepared`
5. `ensureActiveBackupLease`
6. `ensureMaintenanceActivated`
7. `ensureProviderBackupCreated`
8. `ensureProviderBackupCompleted`
9. `ensureMaintenanceDeactivated`
10. `ensureBackupLeaseReleased`
11. `ensureBackupRunCompleted`

Once the provider has produced a terminal result or the backup has been canceled, the controller uses the finalize path containing the last three stages. During deletion, `ensureMaintenanceDeactivated`, `ensureBackupLeaseReleased`, and `ensureProviderBackupDeleted` run. An already terminal backup with `Succeeded=True` or `Succeeded=False` uses the ignore path with `ensureOrphanedBackupDeleted`: the stage checks whether the provider backup still exists and removes an orphaned CES backup CR if necessary. Canceled backups are excluded there and handled by `ensureCanceledProviderBackupDeleted` instead (see [Time window and cancellation](#time-window-and-cancellation)). `Succeeded` is set by `ensureBackupRunCompleted` only after maintenance-mode deactivation and lease release.

## Successful local backup workflow

```mermaid
sequenceDiagram
    participant U as User/Schedule
    participant B as Backup controller
    participant S as BackupStorageLocation
    participant M as Maintenance mode
    participant V as Velero

    U->>B: Create Backup CR
    B->>B: Add metadata and finalizer
    B->>B: Check retryTimeLimit
    B->>S: Read phase
    S-->>B: Available
    B->>M: Activate maintenance mode
    B->>V: Create Velero backup with the same name
    B->>B: Set StartTimestamp
    loop until Velero is terminal
        B->>V: Read backup phase
        V-->>B: New/InProgress/Finalizing/WaitingForPluginOperations
        B->>B: ProviderSucceeded=Unknown, timed retry
    end
    V-->>B: Completed
    B->>B: Set ProviderSucceeded=True and CompletionTimestamp
    B->>M: Deactivate maintenance mode
    B->>B: Release lease
    B->>B: Set Succeeded=True
```

### Setup and metadata

`ensureBackupSetup` sets the following metadata:

- Labels `app=ces` and `k8s.cloudogu.com/part-of=backup`
- Finalizer `cloudogu-backup-finalizer`
- Annotation `backup.cloudogu.com/blueprintId` containing the display name of the first blueprint in the namespace
- Annotation `backup.cloudogu.com/dogus` containing its Dogu list as JSON

If no blueprint exists, the workflow continues without blueprint annotations. Other errors while listing or serializing abort the reconciliation with an error. Existing foreign labels, annotations, and finalizers are preserved; managed values are corrected when they differ.

### Time window and cancellation

The `k8s-backup-operator-backup-config` ConfigMap must contain the `retryTimeLimit` key in the backup namespace. Its value is an integer in minutes. The time window has expired when:

```text
now - metadata.creationTimestamp > retryTimeLimit * time.Minute
```

```mermaid
stateDiagram-v2
    [*] --> TimeWindowNotExpired
    TimeWindowNotExpired --> TimeWindowExpiredBackupNotStarted: Time expired, StartTimestamp empty
    TimeWindowNotExpired --> TimeWindowExpiredBackupInProgress: Time expired, provider running
    TimeWindowNotExpired --> TimeWindowExpiredBackupFailed: Time expired, provider failed
    TimeWindowNotExpired --> TimeWindowExpiredBackupSucceeded: Time expired, provider succeeded
    TimeWindowExpiredBackupNotStarted --> [*]: Canceled=True, no provider backup
    TimeWindowExpiredBackupFailed --> [*]: Canceled=True
    TimeWindowExpiredBackupInProgress --> [*]: Canceled=True, provider backup orphaned
    TimeWindowExpiredBackupSucceeded --> ProviderObservation: Canceled=False
```

If the ConfigMap or key is missing, or if the value is not numeric, the reconciliation ends with an error. `StartTimestamp` separates “not started yet” from “already started.”

#### Canceling a running provider backup

Velero provides no way to cancel a running backup from the outside. The run is therefore abandoned rather than stopped: `Canceled=True` routes the next pass into the finalize path, which deactivates maintenance mode, releases the lease, and writes the terminal `Succeeded=False`. The Velero backup keeps running as an orphan. Holding maintenance mode and the lease for hours past the time window is the worse outcome.

`ensureCanceledProviderBackupDeleted` deletes that orphan once it is no longer in a running phase. Maintenance mode was switched off while Velero may still have been reading, so the result is potentially inconsistent and must not be restorable – not even from another cluster that shares the `BackupStorageLocation`. The backup CR itself is kept as failure history, so both `ensureOrphanedBackupDeleted` and the synchronization controller skip canceled backups.

The deletion runs on the already terminal CR: `ensureBackupRunCompleted` returns `Retry` for canceled runs, and the shared deletion helpers write their progress with `Deleting=False`, because only the provider backup is being deleted, not the CR.

### Preparation

`ensureBackupIsPrepared` reads the configured Velero `BackupStorageLocation`. Only `status.phase=Available` results in `Prepared=True`. A missing or unavailable location results in `Prepared=False` and a controlled retry. Other API errors are returned.

The stage additionally lists the Velero backups of the namespace and blocks with `Prepared=False`/`OtherProviderBackupInProgress` while a Velero backup of another run is still in a running phase. This prevents a new backup from colliding with the orphan of a canceled run. The guard runs before the lease and maintenance mode, so waiting has no user impact – but a backup started shortly after a canceled one can run into its own time window and be canceled as well.

### Shared backup/restore lease

After successful preparation and before maintenance mode is activated, `ensureActiveBackupLease` acquires the namespace-wide Kubernetes `Lease` named `k8s-backup-operator-lease`. The restore controller uses the same lease. This prevents backup and restore from executing their critical sections concurrently in the same namespace.

The holder is described by `spec.holderIdentity` (UID), `k8s.cloudogu.com/backup-operator-lease-holder-name` (name), and `k8s.cloudogu.com/lease-holder-kind` (`Backup` or `Restore`). All three fields are written together and must be present; incomplete leases are considered invalid and are not repaired heuristically. Each controller only registers the resolver for its own resource type. A foreign or unknown holder type is considered active and is not taken over, so the two workflows do not need to know about each other. A resource's own lease is accepted idempotently. The passage of time alone does not make a lease stale.

`ensureBackupLeaseReleased` runs in the finalize and delete paths after `ensureMaintenanceDeactivated`. The stage deletes only the resource's own lease when the backup run is terminal, canceled, or marked for deletion. It is a no-op for running backups and foreign holders. UID and `resourceVersion` preconditions prevent deleting a lease that has since been assigned to another resource. The stage deliberately does not change maintenance mode; `ensureMaintenanceDeactivated` remains responsible for that.

### Maintenance mode

Maintenance mode must be active before the Velero backup is created. The controller activates it with:

```text
Title: Service temporary unavailable
Text:  Backup in progress
force: false
```

After a terminal successful or failed provider backup, maintenance mode is deactivated if it is still active. Activation and deactivation are not best-effort; errors are returned.

The order is relevant: a terminal provider result moves the backup into the finalize path. That path first deactivates maintenance mode, then releases the lease, and only afterward sets `Succeeded` through `ensureBackupRunCompleted`. If deactivation fails, the backup is therefore not incorrectly marked as fully completed and the finalize path runs again.

### Generated Velero backup

The operator creates a Velero backup with the same name and the following settings:

- Exactly the namespace of the backup CR
- Storage location from the operator configuration
- TTL `87660h`, approximately ten years
- `defaultVolumesToFsBackup=false`
- CES standard labels and forwarded backup annotations

Included resource types:

- `configmaps`
- `secrets`
- `persistentvolumeclaims`
- `persistentvolumes`
- `dogus.k8s.cloudogu.com`

The selection is an OR combination of:

1. `k8s.cloudogu.com/type=global-config`
2. Presence of the `dogu.name` label
3. Presence of the `k8s.cloudogu.com/backup-scope` label

The values of `dogu.name` and `backup-scope` are not evaluated for selection. Additional resources are included only if their resource type is also in the include list above.

### Provider phases

| Category | Velero phases | Result |
|---|---|---|
| running | `New`, `InProgress`, `Finalizing`, `FinalizingPartiallyFailed`, `WaitingForPluginOperations`, `WaitingForPluginOperationsPartiallyFailed` | `ProviderSucceeded=Unknown/ProviderBackupInProgress`, retry |
| failed | `FailedValidation`, `PartiallyFailed`, `Failed` | `ProviderSucceeded=False/ProviderBackupFailed` |
| successful | `Completed` | `ProviderSucceeded=True/ProviderBackupSucceeded` |
| unexpected | for example `Deleting` | Error; no implicit success |

When the provider backup is created, `StartTimestamp` is set only if it is still empty. For a terminal result, `CompletionTimestamp` is likewise set only once.

## Conditions

Local backups and backups imported from the provider use the same five conditions:

### `Deleting`

| Status | Reason | Meaning |
|---|---|---|
| `False` | `BackupNotDeleting` | No `deletionTimestamp`; the normal workflow may run. |
| `True` | `BackupDeleting` | Deletion was requested and the provider backup still exists. |
| `False` | `CanceledProviderBackupDeleted` | The orphaned provider backup of a canceled run is gone. |

While the provider backup of a canceled run is deleted, the deletion reasons are written with status `False`: the CR is kept.

### `Canceled`

| Status | Reason | Meaning |
|---|---|---|
| `False` | `TimeWindowNotExpired` | The start time window is still open; the backup was not canceled. |
| `True` | `TimeWindowExpiredBackupNotStarted` | The backup was not started before the window expired. |
| `True` | `TimeWindowExpiredBackupInProgress` | The backup was still running when the window expired; its provider backup is orphaned and gets deleted. |
| `True` | `TimeWindowExpiredBackupFailed` | The provider had already failed when the window expired. |
| `False` | `TimeWindowExpiredBackupSucceeded` | The provider had already succeeded when the window expired. |

### `Prepared`

| Status | Reason | Meaning |
|---|---|---|
| `False` | `ProviderBackupStorageLocationNotFound` | The `BackupStorageLocation` is missing. |
| `False` | `ProviderBackupStorageLocationNotAvailable` | The location exists but is not `Available`. |
| `False` | `OtherProviderBackupInProgress` | A provider backup of another run is still running. |
| `True` | `ProviderBackupStorageLocationAvailable` | Provider storage is usable. |
| `True` | `VeleroStatusSynced` | The imported backup already exists in Velero. |

### `ProviderSucceeded`

| Status | Reason | Meaning |
|---|---|---|
| `Unknown` | `ProviderBackupInProgress` | The provider is working; the create pipeline is retried on a timer. |
| `Unknown` | `VeleroBackupRunning` | An imported Velero backup is not terminal yet. |
| `False` | `ProviderBackupFailed` | The provider failed terminally; the finalize pipeline takes over. |
| `False` | `VeleroBackupFailed` | An imported Velero backup reports a known failure phase. |
| `True` | `ProviderBackupSucceeded` | The provider completed successfully; the finalize pipeline takes over. |
| `True` | `VeleroStatusSynced` | An imported Velero backup is `Completed`. |

### `Succeeded`

| Status | Reason | Meaning |
|---|---|---|
| `Unknown` | `MaintenanceModesIsNotActive` | Maintenance mode activation was initiated, but the mode is not active yet; the workflow continues. |
| `Unknown` | `ProviderBackupResourceDoesNotExist` | The provider child was just created. |
| `Unknown` | `VeleroBackupRunning` | An imported Velero backup is not terminal yet; unknown phases are also considered running. |
| `False` | `ProviderBackupFailed` | The provider failed terminally. |
| `False` | `VeleroBackupFailed` | An imported Velero backup reports a known failure phase. |
| `True` | `ProviderBackupSucceeded` | The provider completed successfully. |
| `True` | `VeleroStatusSynced` | An imported Velero backup is `Completed`. |

`Succeeded=True` and `Succeeded=False` are classified as terminal by `requiredOperation` and routed to the ignore path. The name `MaintenanceModesIsNotActive` historically describes the state before successful activation.

For `spec.syncedFromProvider=true`, `ensureVeleroStatusSynced` also writes the observed state to `Succeeded`. `WaitingForPluginOperationsPartiallyFailed` and `FinalizingPartiallyFailed` remain non-terminal because Velero is still performing plugin or finalization work; only the later terminal `PartiallyFailed` phase sets `Succeeded=False`. Velero timestamps are mirrored as well. As with local backups, the legacy `status.status` field is derived centrally from the conditions by the `conditionsUpdater`.

## Velero backup synchronization

The synchronization controller ensures that the provider and CES catalogs converge:

```mermaid
flowchart TD
    A[Velero event] --> B{Velero backup exists?}
    B -- yes --> C{deletionTimestamp set?}
    C -- no --> D{Backup CR with the same name exists?}
    D -- no --> E[Create Backup CR with syncedFromProvider=true]
    D -- yes --> F[Do nothing]
    C -- yes --> G[Ensure DeleteBackupRequest]
    G --> H{Backup CR canceled?}
    H -- no --> I[Delete Backup CR]
    H -- yes --> J[Keep Backup CR as failure history]
    B -- no --> G
```

An imported backup CR receives provider `velero`, CES standard labels, forwarded annotations, and the backup finalizer. Because Kubernetes does not preserve status from the object during creation, `spec.syncedFromProvider=true` signals the main controller to read the status from Velero afterward.

When a Velero backup is deleted or is already missing, the controller first idempotently ensures a `DeleteBackupRequest` with the same name and then deletes the CES backup CR – unless that CR is canceled, which is kept as failure history. The delete request prevents Velero from later reconstructing the CR from backup data that is still present in storage.

## Delete workflow of a Backup CR

```mermaid
sequenceDiagram
    participant U as User
    participant B as Backup controller
    participant D as DeleteBackupRequest
    participant V as Velero backup
    participant K as Kubernetes

    U->>K: Delete Backup CR

    loop While the provider backup exists
        B->>V: Read provider backup
        V-->>B: Provider backup exists
        alt Provider backup is still running
            B->>D: Remove existing DeleteBackupRequest
            B->>B: Deleting=True, WaitingForProviderBackupCompletion
        else Provider backup is terminal
            B->>D: Idempotently ensure DeleteBackupRequest
            B->>B: Deleting=True, BackupDeleting
        end
        B-->>B: Timed retry
    end

    B->>V: Read provider backup again
    V-->>B: Provider backup is missing
    B->>K: Remove cloudogu-backup-finalizer
    K-->>U: Parent may disappear
```

For running Velero phases (`New`, `InProgress`, `Finalizing`, and the phases for pending plugin operations), the controller does not create a delete request yet. It removes an existing delete request so that Velero can finish the running backup normally. The `Deleting=True` condition with reason `WaitingForProviderBackupCompletion` makes this waiting state visible. The delete request is ensured only after the provider backup is no longer running.

The controller does not wait for the status of the delete request. Instead, it checks during every retry whether the Velero backup still exists. The backup finalizer is removed from the finalizer list and the parent is updated only after the provider backup has disappeared. Other finalizers are preserved.

## Status persistence and idempotency

Condition changes are written centrally by the `conditionsUpdater` using a status `MergeFrom` patch. The legacy `status.status` field used by external systems is synchronized for local and imported backups as well:

| Condition state | `status.status` |
|---|---|
| Running workflow or non-terminal condition | `in progress` |
| `Succeeded=True` | `completed` |
| `Succeeded=False` or `Canceled=True` | `failed` |
| `Deleting=True` | `deleting` |
| No condition yet | Previous value; empty for a new backup |

`Deleting=True` takes precedence over terminal success or failure conditions.

`meta.SetStatusCondition` preserves the existing transition time when the status is unchanged.
Status and conditions are updated only when they actually differ.

## Acceptance tests

The cluster tests are located in `acceptance-tests/backup_test.go` and use the Ginkgo label `backup`:

```bash
K8S_TEST_CLUSTER_KUBECONFIG=/absolute/path/to/kubeconfig \
  make acceptance-test GINKGO_LABEL_FILTER=backup
```
