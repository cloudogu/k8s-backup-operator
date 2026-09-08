# Metrics

The operator provides counters for reconciliations as well as for persisted status and condition transitions of backups and restores.

| Metric | Labels | Meaning |
|---|---|---|
| `backup_reconcile_total` | – | Total number of backup reconciliations. |
| `restore_reconcile_total` | – | Total number of restore reconciliations. |
| `backup_status_transitions_total` | `namespace`, `name`, `to` | Persisted changes of the deprecated scalar backup status. |
| `restore_status_transitions_total` | `namespace`, `name`, `to`, `backup_name` | Persisted changes of the deprecated scalar restore status. |
| `backup_condition_transitions_total` | `namespace`, `name`, `condition`, `from`, `to` | Persisted status changes of a backup condition. |
| `restore_condition_transitions_total` | `namespace`, `name`, `backup_name`, `condition`, `from`, `to` | Persisted status changes of a restore condition. |

## Condition transitions

The condition metrics count only actual transitions between `Unknown`, `True`, and `False`, for example `Prepared: Unknown -> True`. The initial creation of a condition and changes limited to `Reason`, `Message`, `ObservedGeneration`, or the transition time are not counted.

The respective condition updater increments the counter only after a successful status write. Failed writes and discarded conflict attempts therefore do not appear in the metric. After a conflict, the transitions are recalculated against the resource status that was read again.

After reading a resource, each reconciliation initializes all six possible directed status transitions with `Add(0)`. This makes transitions that have never been observed available as zero-valued time series as well:

| Resource | Conditions | Time series per resource |
|---|---|---|
| Backup | `Prepared`, `Deleting`, `Canceled`, `Succeeded`, `ProviderSucceeded` | 5 × 6 = 30 |
| Restore | `Succeeded`, `Prepared`, `ProviderSucceeded`, `WorkloadsRecovered` | 4 × 6 = 24 |

Example for successfully reached milestones:

```promql
sum by (condition) (
  increase(backup_condition_transitions_total{from="Unknown", to="True"}[1h])
)
```

For restores, use `restore_condition_transitions_total` accordingly. The additional `backup_name` label can be used to group or filter by the restored backup.

## Legacy status and reconciliations

The `backup_status_transitions_total` and `restore_status_transitions_total` metrics remain available for the scalar compatibility status `status.status`. New queries should preferably use the more detailed condition transitions.

`backup_reconcile_total` and `restore_reconcile_total` count every invocation of the respective reconciler, regardless of the outcome. Like all Prometheus counters, their values start at zero again after the operator restarts.
