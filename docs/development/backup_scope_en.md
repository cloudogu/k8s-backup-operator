# Backup Scope

Currently, backup of single Dogus is not yet supported.
For now, the data of all Dogus and the global config is backed up.
Specifically, the resources with the label key `dogu.name` and the `k8s.cloudogu.com/type: global-config` label are selected.
Only ConfigMaps, Secrets, PersistentVolumeClaims and the Dogu resource itself are backed up.
Because the Dogu resource is backed up, the Dogu operator can recreate any other resources that are not included in the backup.

To back up other resources, e.g., PersistentVolumeClaims from components, a label with the key `k8s.cloudogu.com/backup-scope` can be added to the resource.
Be aware that the limitation to the aforementioned kinds of resources still applies.