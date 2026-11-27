# Restore Process

A CES restore restores a CES backup into the namespace. The restore controller determines which backup units of the
backup to restore and creates an associated Velero restore for each one. `ProviderSucceeded` summarizes the
results across those units. With `spec.excludedDogus`, `spec.excludedComponents` and options, the restore can be applied 
only in parts. A legacy backup without units is restored as a whole by a single Velero restore with the same name of the 
CES restore.

The controller works in stages: each reconciliation runs the stages in order until one of them writes the restore, has
to wait, or fails. Conditions record progress, but the cluster resources stay the source of truth, so a restarted
operator resumes where the previous run stopped. Unlike the backup controller, the restore controller also watches its
own Velero restores. Their phase changes trigger the next reconciliation directly. Timed requeues are after the creation 
of the velero resources only needed as a safety net.

## Restore Workflow

1. The controller initializes the conditions `Succeeded`, `Prepared`, `ProviderSucceeded` and `WorkloadsRecovered`
   as `Unknown`, adds its finalizer and labels, and acquires the [shared lease for backup and restore](parallel_access_en.md).
2. Before anything destructive happens, it checks that Velero is ready and that the backup can be restored as requested.
   This includes checking that the backup exists, the exclusions match it, and the backup units being restored are usable.
   It also checks that no Velero restore with one of the expected names belongs to someone else. A failed check ends the
   restore with `Succeeded=False`.
3. It activates maintenance mode (best effort), scales down the workloads and deletes the resources that are about to be
   restored (`Prepared=True`).
4. It creates the Velero restores of the units and waits for their results (`ProviderSucceeded`).
5. It scales the workloads back up, waits until they are ready, removes the temporary replica labels and deactivates
   maintenance mode. Only then does it set `WorkloadsRecovered=True` and `Succeeded=True`, which releases the lease.

Preparation runs only once: if `Prepared=True` is set or a Velero restore owned by this restore exists, the
preparation steps and checks are skipped.

## Failure and Deletion

If Velero fails the restore of a unit, the controller sets `Succeeded=False` and deactivates maintenance mode. It
deliberately leaves the workloads scaled down, because the data they would start on is incomplete. Since
`Succeeded=False` is terminal, the lease is released and another backup or restore can start. Errors in other steps
are retried with backoff.

A deletion request takes precedence over the workflow. The controller deletes its own Velero restores and waits until
they are actually gone, releases the lease and then removes its finalizer. Velero restores that belong to someone else
are left in place and do not block deletion.

Restores created by older operator versions only have the deprecated `status.status` field. The controller converts it
into a `Succeeded` condition once and keeps writing `status.status` for older clients.

## Code Entry Points

- The [restore reconciler](../../internal/controller/restore/restoreReconciler.go) defines the stage order for creation
  and deletion; [stageOutcome.go](../../internal/controller/restore/stageOutcome.go) defines how a stage continues,
  waits or fails.
- The [restore state](../../internal/controller/restore/state) collects the backup units of the source backup and
  decides which units to restore.
- The [Velero restore logic](../../internal/provider/velero/restore.go) creates, observes and deletes the Velero restores.
- [scale.go](../../internal/controller/restore/scale.go) scales the workloads down and up again. And 
[cleanup.go](../../internal/cleanup/cleanup.go) handles the deletion of the resources to be restored.

## Acceptance Tests

The restore specs in [`specs`](../../specs) are destructive: they clean up the `ecosystem` namespace. Run them only
against a disposable cluster, with `K8S_TEST_CLUSTER_KUBECONFIG` pointing to exactly one kubeconfig file:

```bash
K8S_TEST_CLUSTER_KUBECONFIG=/absolute/path/to/kubeconfig make acceptance-test-restore
```

`GINKGO_FOCUS='<regex>'` runs a single spec.
