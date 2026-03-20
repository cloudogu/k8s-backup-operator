# Velero

Bei der Installation mit ecosystem-core wird Velero automatisch installiert, wenn das flag `backup=true` gesetzt ist.
Ansonsten muss Velero manuell installiert werden. Dafür kann folgende yaml verwendet werden. Der Serviceaccount und der
Bucket müssen angepasst werden.

## Velero-Secret erstellen
Für den Zugriff auf den Storage-Bucket muss ein Schlüssel erstellt werden. Am einfachsten wird dieser Schlüssel über
die Google-Cloud-Oberfläche erstellt.
- Google-Cloud-Weboberfläche aufrufen
- zu Serviceaccounts navigieren
    - IAM & Admin -> Service Accounts -> <my-service-account> -> Keys -> Add key -> Create new key
        - Typ: JSON
        - Create
    - Der Schlüssel wird automatisch als JSON-Datei heruntergeladen
- Schlüssel als Secret im Cluster speichern

```yaml
kubectl create secret generic -n ecosystem velero-backup-target --from-file=cloud=<keyfile.json>
```

## Velero installieren

### backupStorageLocation und volumeSnapshotLocation

Bei der Erstellung von velero müssen die backupStorageLocation und volumeSnapshotLocation angegeben werden.
Die ``backupStorageLocation`` bestimmt, wo die Metadaten des Velero-Backups abgelegt werden. Dies ist immer ein S3-Bucket.
Die ``volumeSnapshotLocation`` bestimmt, wo die Snapshots der Volumes abgelegt werden. Je nachdem, welcher CSI verwendet wird,
werden die Daten an anderen Orten gespeichert. In der Google Cloud werden Volumesnapshots angelegt. Wird ``longhorn`` verwendet,
wird hier ein weiterer S3-Bucket benötigt. Außerdem können abhängig vom verwendeten CSI auch andere Velero-Plugins benötigt werden.

### Velero-Konfiguration (Beispiel für Google Cloud)
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
              serviceAccount: "<my-service-account-email>"
        volumeSnapshotLocation:
          - name: default
            provider: velero.io/gcp
            config:
              snapshotLocation: europe-west3
```

Die Datei kann mit gesetztem Kubecontext mit `kubectl apply -f velero.yaml -n ecosystem` angewendet werden.

**weitere Beispiele**
* [Google Cloud Bucket einrichten](./setup_gcloud_backup_bucket_de.md)
* [Longhorn einrichten](./use_longhorn_storage_provisioner_de.md)

## VolumeSnapshotClass anpassen
Die vorhandene Volumesnapshotclass muss erweitert werden. Die Labels werden benötigt, um die Snapshots den Backups 
zuzuordnen.
```yaml
...
parameters:
  storage-locations: europe-west3
  labels: team=ces,purpose=mn-testing-cluster-backup,backup=mn-testing-cluster-backup
```

## Installation validieren

Wenn die Installation erfolgreich war, dann sollte folgende Ausgabe in Velero angezeigt werden: `BackupstorageLocation is valid, marking as available`