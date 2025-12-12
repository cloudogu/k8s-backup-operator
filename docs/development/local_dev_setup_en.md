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

You can use the configuration in `../../samples/setup/additionalValues.yaml` to install and configure the Backup/Restore stack
inside a CES test cluster by executing the following commands:

```shell
helm get values ecosystem-core -o yaml -n ecosystem > original.yaml

if yq --version 2>&1 | grep -qi "mikefarah"; then
  # mikefarah version of yq installed
  yq eval-all 'select(fi==0) * select(fi==1)' original.yaml ../../samples/setup/additionalValues.yaml > merge.yaml
else
  # kislyuk version of yq installed
  yq -y --sort-keys '. *= input' original.yaml ../../samples/setup/additionalValues.yaml > merge.yaml
fi

helm upgrade ecosystem-core oci://registry.cloudogu.com/k8s/ecosystem-core --version 2.0.2 -n ecosystem -f merge.yaml
```

Additionally, Longhorn needs to be configured:

```shell
helm get values longhorn -o yaml -n longhorn-system > longhorn_original.yaml

if yq --version 2>&1 | grep -qi "mikefarah"; then
  # mikefarah version of yq installed
  yq eval-all 'select(fi==0) * select(fi==1)' longhorn_original.yaml ../../samples/setup/longhornAdditionalValues.yaml > longhorn_merge.yaml
else
  # kislyuk version of yq installed
  yq -y --sort-keys '. *= input' longhorn_original.yaml ../../samples/setup/longhornAdditionalValues.yaml > longhorn_merge.yaml
fi

helm upgrade longhorn longhorn/longhorn --version 1.10.0 -n longhorn-system -f longhorn_merge.yaml
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
