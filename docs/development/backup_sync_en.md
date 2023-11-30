# Backup Synchronization

After a restore was applied to the Cloudogu EcoSystem (CES), some of the Velero backups may have no representation
as Backup Custom Resources (CRs) anymore. This applies to Velero backups which have been created after the backup that
was restored.

![Backup synchronization problem after restore](restore_problem.png "Backup synchronization problem after restore")

The solution to this is the synchronization of the Velero backups with the Backup CRs after a restore has been applied.
This process will create Backup CRs for Velero backups which are missing in the cluster. It will also delete Backup CRs,
which have no corresponding Velero backup.