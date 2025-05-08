# k8s-backup-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Fixed
- [#54] Restore of the velero deployment disrupted the restore
  - Previously, the Velero deployment would get restored as well,
    which caused disruptions of the restore if the deployment is different
    from the one in the backup.
- [#56] Cleanup of backup components lead to errors after restore
  - This is because the component operator would detect a downgrade, which is not allowed.
    Or worse, an upgrade during the restore operation would cause it to fail.
### Changed
- [#54] Exclude all resources with label `k8s.cloudogu.com/part-of: backup` from restores.
- [#56] Exclude all backup-related components from cleanup when restoring.

## [v1.4.0] - 2025-05-07
### Added
- [#50] Exclude resources in cleanup before restore

## [v1.3.4] - 2025-04-30

### Changed
- [#51] Set sensible resource requests and limits

## [v1.3.3] - 2025-04-10
### Fixed
- [#44] Fixed resource ownerReferences after a restore of a previous backup
  - subsequently, this fixes the failure to fully delete some resources after a restore

## [v1.3.2] - 2025-04-03
### Fixed
- [#46] Fixed endless loop of reconciles because the image was not set in update manager and difference comparison 
  did not check the cron job provider correctly
- [#46] Fixed missing CronJobs apiGroup in RBACs
- [#46] Add a waiting routine to fix race condition while deleting and restoring resources

### Added
- [#47] Add additional print columns and aliases to CRDs

## [v1.3.1] - 2024-12-19
### Fixed
- [#42] Removed unnecessary rbac proxy to fix CVE-2024-45337

## [v1.3.0] - 2024-12-05
### Added
- [#40] Add NetworkPolicy to deny all ingress traffic

## [v1.2.0] - 2024-11-29
### Changed
- [#37] Refactor rbac permissions to be more clear and better match the use cases

### Removed
- [#37] Leader election and leader election rbac permissions
- [#37] Metrics rbac permissions

### Fixed
- Do not abort restore when maintenance mode cannot be activated

## [v1.1.1] - 2024-10-29
### Fixed
- [#35] Use correct helm dependency constraint for `backup-operator-crd`.

## [v1.1.0] - 2024-10-28
### Changed
- [#33] Make imagePullSecrets configurable via helm values and use `ces-container-registries` as default.

## [v1.0.0] - 2024-10-18
### Changed
- [#31] Use cluster native config instead of the etcd.

## [v0.11.0] - 2024-09-18
### Changed
- [#29] Relicense to AGPL-3.0-only

## [v0.10.1] - 2024-01-10
### Fixed
- [#27] Added missing watch permission for statefulsets.
  - This is used when waiting for the etcd on maintenance switch.

### Changed
- [#14] Updated docs for installing and configuring `k8s-longhorn` and `k8s-velero`.

## [v0.10.0] - 2023-12-19
### Added
- [#23] Added docs for installing the operator in an existent Cloudogu EcoSystem and on an empty cluster.

### Fixed
- Fix value of label `k8s.cloudogu.com/part-of` in helm template do avoid deletion in cleanup process.

## [v0.9.0] - 2023-12-08
### Added
- [#14] Patch template for mirroring this component and its images
### Changed
- [#17] Replace create-backup-script with an operator subcommand.
  This way, the backup schedule cron job can use the same image as the operator.
- [#15] Delete kustomize structure and hold the operator yaml files just in a helm chart.

## [v0.8.0] - 2023-12-04
### Added
- [#19] Sync backup list with provider on operator startup

## [v0.7.0] - 2023-11-30
### Added
- [#13] Sync completed (velero) backups with backup CRs after a restore has finished

## [v0.6.0] - 2023-11-22
### Added
- [#11] Automated deletion of Backups via various retention strategies.

## [v0.5.0] - 2023-11-09
### Added
- [#8] Functionality to schedule backups via a `BackupSchedule` Resource

## [v0.4.0] - 2023-10-25
### Added
- [#7] Functionality to restore a backup to the namespace where the backup-operator is deployed
    - Before the restore is applied, resources in this namespace which are irrelevant to the backup process are removed to provide a clean slate
    - Currently, only the velero provider is supported

## [v0.3.0] - 2023-10-06
### Added
- [#3] Functionality to create a backup from the namespace where the backup-operator is deployed
  - Velero is used as a first provider.

## [v0.2.0] - 2023-10-05
### Added
- [#4] Add CRD-Release to Jenkinsfile

## [v0.1.0] - 2023-09-05

Initial release

