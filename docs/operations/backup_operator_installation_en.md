# Installing the backup operator

The backup operator can usually be installed in an existing Cloudogu EcoSystem or empty cluster.

## Installation with an existing Cloudogu EcoSystem

In an existing Cloudogu EcoSystem, the backup operator is installed via the component operator.
To do this, a custom resource `Component` must be created for the backup operator and its own CRDs.

### Dependencies

However, the operator's dependencies should be installed first. The backup operator requires a backup provider.
Currently `velero` is supported as a provider.
If no snapshot API is available in the cluster, a snapshot controller must also be installed.
The same applies to the storage provider.

#### Backup storage

The backups are stored in an S3-compatible object storage, e.g. [Minio](https://min.io/).
This storage should be located outside the Kubernetes cluster, so that the backups are still available and secure if the cluster fails.
The installation and operation of the backup storage must therefore be carried out separately from the CES.

### Storage provisioner

If no storage provisioner exists in the cluster, `longhorn` can be installed and used.

#### Create a secret for the backup storage

Longhorn backups are stored in the backup storage described above. To do this, `longhorn` needs access to the storage.
The parameters required for this must be stored in a Kubernetes secret:

| Secret Key            | Description                               |
|-----------------------|-------------------------------------------|
| AWS_ENDPOINTS         | The URL of the backup storage             |
| AWS_ACCESS_KEY_ID     | The ID of the AccessKey for Longhorn      |
| AWS_SECRET_ACCESS_KEY | The secret for the AccessKey for Longhorn |

The secret can be created with the following example command:

```shell
kubectl create secret generic longhorn-backup-target --namespace=longhorn-system \
--from-literal=AWS_ENDPOINTS=https://192.168.56.1:9000 \
--from-literal=AWS_ACCESS_KEY_ID=MY-ACCESS-KEY \ 
--from-literal=AWS_SECRET_ACCESS_KEY=MY-ACCESS-SECRET123
```

The secret must be created in the same Kubernetes namespace as `longhorn`.

#### Configure Longhorn

The attribute `valuesYamlOverwrite` can be used to configure the URL and credentials for backups to an S3 storage.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-longhorn
spec:
  name: k8s-longhorn
  deployNamespace: longhorn-system
  namespace: k8s
  valuesYamlOverwrite: |
    longhorn:
      defaultSettings:
        backupTarget: s3://longhorn@dummyregion/
        backupTargetCredentialSecret: long-backup-target
```

The following parameters in the `valuesYamlOverwrite` are relevant for the backup:

| Parameter                                               | Description                                                                                           |
|---------------------------------------------------------|-------------------------------------------------------------------------------------------------------|
| `longhorn.defaultSettings.backupTarget`                 | The address of the storage location (bucket) within the backup storage: `s3://<BUCKET_NAME>@<REGION>` |
| `longhorn.defaultSettings.backupTargetCredentialSecret` | The name of the secret created above that contains the access data to the backup storage              |

The `yaml` file created for the Longhorn component can be used with the following command:

`kubectl --namespace ecosystem apply -f k8s-longhorn.yaml`

#### Snapshot API

If the Kubernetes cluster does not support the snapshot API, a snapshot controller must also be installed.
This is the case if, for example, `k3s` is used as the Kubernetes distribution.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-snapshot-controller-crd
spec:
  name: k8s-snapshot-controller-crd
  namespace: k8s
```

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-snapshot-controller
spec:
  name: k8s-snapshot-controller
  namespace: k8s
```

Installation:

`kubectl --namespace ecosystem apply -f k8s-snapshot-controller-crd.yaml`

`kubectl --namespace ecosystem apply -f k8s-snapshot-controller.yaml`

#### Velero

Velero also requires configuration to store the backups.

#### Create a secret for the backup storage

Velero backups are also stored in the backup storage described above. Velero needs access to the storage for this.
The parameters required for this must be stored in a Kubernetes secret:

| Secret Key            | Description                             |
|-----------------------|-----------------------------------------|
| aws_access_key_id     | The ID of the AccessKey for Velero      |
| aws_secret_access_key | The secret for the AccessKey for Velero |

The secret for is used as a file in Velero and must therefore be created according to the following example:

```shell
kubectl apply --namespace=ecosystem -f - <<EOF
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: velero-backup-target
stringData:
  cloud: |
    [default]
    aws_access_key_id=MY-VELERO-ACCESS-KEY
    aws_secret_access_key=MY-VELERO.ACCESS-SECRET123
EOF
```

The secret must be created in the same Kubernetes namespace as `velero`.

#### Configure Velero

The attribute `valuesYamlOverwrite` can also be used here to add or overwrite any configurations:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-velero
spec:
  name: k8s-velero
  namespace: k8s
  valuesYamlOverwrite: |
    velero:
      credentials:
        useSecret: true
        existingSecret: "velero-backup-target" # Name of a pre-existing secret in the Velero namespace
      configuration:
        backupStorageLocation:
          - name: default
            provider: aws
            bucket: velero # Ensure that this bucket exists in the Storage. Furthermore, if you use longhorn the bucket `longhorn` has to be created.
            accessMode: ReadWrite
            config:
              region: minio-default
              s3ForcePathStyle: true
              s3Url: http://192.168.56.1:9001 # Insert your url here
              publicUrl: http://localhost:9001 # Insert your url here
```

The following parameters in the `valuesYamlOverwrite` are relevant for the backup configuration:

| Parameter                                                              | Description                                                                                         |
|------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| `velero.credentials.existingSecret`                                    | The name of the secret created above that contains the access data for Velero to the backup storage |
| `velero.configuration.backupStorageLocation[default].bucket`           | The name of the bucket for Velero within the backup storage                                         |
| `velero.configuration.backupStorageLocation[default].config.s3Url`     | The URL of the backup storage                                                                       |
| `velero.configuration.backupStorageLocation[default].config.publicUrl` | The public URL of the backup storage                                                                |

The created `yaml` file for the Velero component can be applied with the following command:

```shell
kubectl --namespace ecosystem apply -f k8s-velero.yaml
```

### Installation backup operator

The backup operator can then be installed with its Component-CRs:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator-crd
spec:
  name: k8s-backup-operator-crd
  namespace: k8s
```

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator
spec:
  name: k8s-backup-operator
  namespace: k8s
```

`kubectl --namespace ecosystem apply -f k8s-backup-operator-crd.yaml`

`kubectl --namespace ecosystem apply -f k8s-backup-operator.yaml`


---
> Info:
>
> The versions of the components can be customized using the `version` attribute:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator
spec:
  name: k8s-backup-operator
  namespace: k8s
  version: 0.1.0
```


If all components have the status `RUNNING`, you can check whether the BackupStorageLocation is available and the S3 storage of Velero is accessible.

`kubectl --namespace ecosystem get backupstoragelocation default`

A regular backup can then be performed. See [Perform backup](backup_en.md).

## Installation in an empty cluster

It may make sense not to restore backups to an existing Cloudogu EcoSystem.
This makes sense, for example, if you want to perform a restore to a new cluster.
This saves an initial setup of the Cloudogu EcoSystem.

The difference to the method [Installation in an existing cluster](#installation-with-existing-cloudogu-ecosystem) is that the installation cannot be carried out with the component operator.
Instead, the regular Helm charts are used. Configurations are not in the component CR, but in a values.yaml.

The determination of dependencies remains the same with this method:
- If no storage provisioner exists, `k8s-longhorn` must be installed.
- If no snapshot API exists, `k8s-snapshot-controller-crd` and `k8s-snapshot-controller` must be installed.
- The backup provider `k8s-velero` must be installed.

### Helm registry login

Since the component operator in an existing cluster has credentials for the Helm registry, this method requires you to authenticate yourself directly with the registry.

`helm registry login registry.cloudogu.com`

### Storage-Provisioner

Creation of the Longhorn secret for the backup storage:

```shell
kubectl create secret generic longhorn-backup-target --namespace=longhorn-system \
--from-literal=AWS_ENDPOINTS=https://192.168.56.1:9000 \
--from-literal=AWS_ACCESS_KEY_ID=MY-ACCESS-KEY \
--from-literal=AWS_SECRET_ACCESS_KEY=MY-ACCESS-SECRET123
```

Configuration k8s-longhorn-values.yaml:

```yaml
longhorn:
  defaultSettings:
    backupTarget: s3://longhorn@dummyregion/
    backupTargetCredentialSecret: long-backup-target
```

Installation:

`helm install k8s-longhorn oci://registry.cloudogu.com/k8s/k8s-longhorn --version 1.5.1-3 -f k8s-longhorn-values.yaml --namespace longhorn-system --create-namespace`

### Snapshot API

Installation:

`helm install k8s-snapshot-controller-crd oci://registry.cloudogu.com/k8s/k8s-snapshot-controller-crd --version 5.0.1-5 --namespace ecosystem --create-namespace`

`helm install k8s-snapshot-controller oci://registry.cloudogu.com/k8s/k8s-snapshot-controller --version 5.0.1-5 --namespace ecosystem`

### Velero

Creation of the Velero secret for the backup storage:

```shell
kubectl apply --namespace=ecosystem -f - <<EOF
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: velero-backup-target
stringData:
  cloud: |
    [default]
    aws_access_key_id=MY-VELERO-ACCESS-KEY
    aws_secret_access_key=MY-VELERO.ACCESS-SECRET123
EOF
```

Configuration values.yaml:

```yaml
velero:
  credentials:
    useSecret: true
    existingSecret: "velero-backup-target" # Name of a pre-existing secret in the Velero namespace
  configuration:
    backupStorageLocation:
      - name: default
        provider: aws
        bucket: velero # Ensure that this bucket exists in the Storage. Furthermore, if you use longhorn the bucket `longhorn` has to be created.
        accessMode: ReadWrite
        config:
          region: minio-default
          s3ForcePathStyle: true
          s3Url: http://192.168.56.1:9001 # Insert your url here
          publicUrl: http://localhost:9001 # Insert your url here
```

The `aws_access_key_id`, `aws_secret_access_key_id`, `s3Url` and `publicUrl` fields must be adapted accordingly.

`helm install k8s-velero oci://registry.cloudogu.com/k8s/k8s-velero --version 5.0.2-4 -f k8s-velero-values.yaml --namespace ecosystem`

### Installation backup operator

The backup operator can then be installed:

`helm install k8s-backup-operator-crd oci://registry.cloudogu.com/k8s/k8s-backup-operator-crd --version 0.9.0 --namespace ecosystem`

`helm install k8s-backup-operator oci://registry.cloudogu.com/k8s/k8s-backup-operator --version 0.9.0 --namespace ecosystem`

If all components have the status `RUNNING`, you can check whether the BackupStorageLocation is available and the S3 storage of Velero is accessible.

`kubectl --namespace ecosystem get backupstoragelocation default`

A regular restore can then be performed. See [Execution Restore](restore_en.md).

### Helm registry logout

`helm registry logout registry.cloudogu.com`.