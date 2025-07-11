# Getting started

Prerequisite for the installation is a running [Multinode EcoSystem][mn-ecosystem-repo].
With the default configuration, only Longhorn is supported as a storage provider in the Multinode EcoSystem.
Theoretically, however, it should also be possible to configure other CSI-capable storage providers.

[mn-ecosystem-repo]: https://github.com/cloudogu/k8s-ecosystem

A S3-compatible object storage is required to be able to save backups.
In this example, MinIO is executed on the host for this purpose:
```shell
../../samples/setup/run_local_minio.sh
```

Secrets are required for communication with the Minio. These can be imported into the cluster as follows:
```shell
../../samples/setup/create_backup_secrets.sh
```

You can log in to the MinIO web interface (http://localhost:9090) with the access data `admin123:admin123`
. Then create two buckets `velero` and `longhorn`. Two access keys are also required:
- Name: `MY-ACCESS-KEY` Secret: `MY-ACCESS-SECRET123`
- Name: `MY-VELERO-ACCESS-KEY` Secret: `MY-VELERO.ACCESS-SECRET123`
Longhorn and Velero are already preconfigured accordingly and therefore do not need to be customised.


The following blueprint provides a basic configuration of the backup stack with all the necessary components:

```shell
kubectl apply -f ../../samples/setup/blueprint_configure_backup.yaml --namespace=ecosystem
```

Before a backup, check whether the backup storage location is accessible:
```shell
kubectl get backupstoragelocation --namespace=ecosystem
```

A backup can then be performed:
```shell
kubectl apply -f ../../samples/backup.yaml --namespace=ecosystem
```