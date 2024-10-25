# k8s-backup-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
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

