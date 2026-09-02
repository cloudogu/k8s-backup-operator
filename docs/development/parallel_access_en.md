# Shared lease for backup and restore

Backup and restore use the same Kubernetes `Lease`, `k8s-backup-operator-lease`, per namespace. This ensures that only one of the two operations can execute its critical section at a time. The generic acquire, resolver, and release logic is located in `internal/leases`; the controllers integrate it into their respective workflows as dedicated stages.

## Holder identity

The lease identifies its holder using three values:

| Field | Content |
|---|---|
| `spec.holderIdentity` | UID of the `Backup` or `Restore` resource |
| `k8s.cloudogu.com/backup-operator-lease-holder-name` | Name of the resource |
| `k8s.cloudogu.com/lease-holder-kind` | `Backup` or `Restore` |

Kind, name, and UID must match for a resource to recognize itself as the holder or to release its own lease.

## Acquisition and waiting

If the lease does not exist, it is created for the current resource. If the resource already holds the lease, its workflow may continue. Otherwise, it waits and retries during a later reconciliation.

Each controller only registers the resolver for its own resource type. A lease held by the other type is therefore considered active until the responsible controller releases it. A waiting restore sets `Succeeded=Unknown/WaitingForActiveRestore` and does not start a destructive stage; a backup returns `Retry` accordingly.

The lease does not become invalid over time and is not renewed periodically while an operation is running. `acquireTime`, `renewTime`, and `leaseTransitions` are updated only when the lease is created or taken over.

## Invalid leases and takeover

UID, name, and kind are always written together. If any of these fields is missing, the lease is considered invalid and is neither reconstructed from other resources nor repaired automatically. A restore records this as `Succeeded=Unknown/InvalidRestoreLease`; errors are retried using the `controller-runtime` backoff.

A structurally complete lease can be taken over if its holder no longer exists, its UID no longer matches the named object, or the holder is in a terminal state. An unknown holder type, or one that the controller cannot resolve, is not taken over for safety reasons.

Lease changes use the Kubernetes `resourceVersion` for optimistic concurrency control. After a conflict, the next reconciliation reads the current state again.

## Release

Backups release their own lease as soon as they succeed, fail, are canceled, or are marked for deletion. Terminal restores release it in the ignore workflow; when a restore is deleted, the lease is released after the provider child has been confirmed as removed and before the parent finalizer is removed.

The release operation verifies kind, name, and UID. UID and `resourceVersion` preconditions on deletion prevent accidentally removing a lease that has since been assigned to another resource.
