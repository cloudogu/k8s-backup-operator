# Exclude resources during a restore

The restore is divided into two steps. Firstly, a cleanup is performed. This cleanup deletes all dogus that are restored.
For a restore from backup units, dogus and components excluded via `excludedDogus`, `excludedComponents` or
`options.ignoreV2Dogus` are kept. For a restore from a legacy backup, all dogus are deleted.

In the second step, the backup provider is used to perform a restore and add all resources from the
backup to the cluster.
