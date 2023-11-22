# k8s-backup-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

