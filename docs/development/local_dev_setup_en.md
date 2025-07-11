# Local dev setup

Prerequisite for the installation is a running [Multinode EcoSystem](https://github.com/cloudogu/k8s-ecosystem).
With the default configuration, only Longhorn is supported as a storage provider in the Multinode EcoSystem.
Theoretically, however, it should also be possible to configure other CSI-capable storage providers.

A S3-compatible object storage is required to be able to save backups.
In this example, MinIO is executed on the host for this purpose:
```shell
../../samples/setup/run_local_minio.sh
```

The MinIO web interface (http://localhost:9090) is accessible. You can log in with `admin123:admin123`


Secrets are required for communication with the Minio. These can be imported into the cluster as follows:
```shell
../../samples/setup/create_backup_secrets.sh
```

The following blueprint provides a basic configuration of the backup stack with all the necessary components:

```shell
kubectl apply -f ../../samples/setup/blueprint_configure_backup.yaml --namespace=ecosystem
```

In order for the `k8s-backup-operator` to be able to communicate with `k8s-longhorn`, the network policies must be removed from the namespace
`longhorn-system`. Otherwise, it will not be possible for the `k8s-backup-operator` to reach the `admission-controller`
from `k8s-longhorn`.

Before a backup, check whether the backup storage location is accessible:
```shell
kubectl get backupstoragelocation --namespace=ecosystem
```

A backup and restore can then be performed:
```shell
kubectl apply -f ../../samples/backup.yaml --namespace=ecosystem
```

```shell
kubectl apply -f ../../samples/restore.yaml --namespace=ecosystem
```