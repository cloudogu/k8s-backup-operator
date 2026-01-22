# Velero

When installing with ecosystem-core, Velero is automatically installed if the flag `backup=true` is set.
Otherwise, Velero must be installed manually. The following yaml can be used for this. The service account and the
bucket must be customized.

## Create Velero secret
A key must be created to access the storage bucket. The easiest way to create this key is via
the Google Cloud interface.
- Open the Google Cloud web interface
- Navigate to service accounts
  - IAM & Admin -> Service Accounts -> <my-service-account> -> Keys -> Add key -> Create new key
    - Type: JSON
    - Create
  - The key is automatically downloaded as a JSON file
- Save the key as a secret in the cluster

```yaml
kubectl create secret generic -n ecosystem velero-backup-target --from-file=cloud=<keyfile.json>
```

## Installing velero

### backupStorageLocation and volumeSnapshotLocation

When creating velero, the backupStorageLocation and volumeSnapshotLocation must be specified.
The ``backupStorageLocation`` determines where the metadata of the Velero backup is stored. This is always an S3 bucket.
The ``volumeSnapshotLocation`` determines where the snapshots of the volumes are stored. Depending on which CSI is used,
the data is stored in different locations. Volume snapshots are created in Google Cloud. If ``longhorn`` is used,
an additional S3 bucket is required here. Depending on the CSI used, other Velero plugins may also be required.

### Velero configuration (example for Google Cloud)
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-velero
  labels:
    app: ces
    app.kubernetes.io/name: k8s-velero
spec:
  name: k8s-velero
  namespace: k8s
  version: 10.0.1-5
  valuesYamlOverwrite: |
    volumesnapshotclass:
      driver: "pd.csi.storage.gke.io"
      parameters:
        type: ""
    velero:
      credentials:
        useSecret: true
        existingSecret: velero-backup-target
      initContainers:
        - name: "velero-plugin-for-gcp"
          image: "velero/velero-plugin-for-gcp:v1.12.1"
          volumeMounts:
            - "mountPath": "/target"
              "name": "plugins"
      configuration:
        backupStorageLocation:
          - name: default
            provider: "velero.io/gcp"
            bucket: "<my-bucket-name>"
            config:
              serviceAccount: "<my-service-account>"
        volumeSnapshotLocation:
          - name: default
            provider: velero.io/gcp
            config:
              snapshotLocation: europe-west3
```

The file can be applied with the Kubecontext set using `kubectl apply -f velero.yaml -n ecosystem`.

**weitere Beispiele**
* [Configure google cloud bucket](./setup_gcloud_backup_bucket_en.md)
* [Set up longhorn](./use_longhorn_storage_provisioner_en.md)

## Customize VolumeSnapshotClass
The existing volume snapshot class must be extended. The labels are required to assign the snapshots to the backups.
The labels are required to assign the snapshots to the backups.

```yaml
...
parameters:
  storage-locations: europe-west3
  labels: team=ces,purpose=mn-testing-cluster-backup,backup=mn-testing-cluster-backup
```

## Validate installation

If the installation was successful, the following output should be displayed in Velero: `BackupstorageLocation is valid, marking as available`