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
--from-literal=AWS_ENDPOINTS=http://192.168.56.1:9000 \
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
        backupTargetCredentialSecret: longhorn-backup-target
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
              s3Url: http://192.168.56.1:9000 # Insert your url here
              publicUrl: http://localhost:9000 # Insert your url here
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