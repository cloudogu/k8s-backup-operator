# Longhorn als Storage-Provisioner

## Longhorn einrichten
Falls im Cluster kein Storage-Provisioner existiert kann `longhorn` installiert und verwendet werden.

### Secret für den Backup-Speicher erstellen

Longhorn-Backups werden im oben beschriebenen Backup-Speicher abgelegt. Dazu benötigt `longhorn` Zugriff auf den Speicher.
Die dafür benötigten Parameter müssen in einem Kubernetes-Secret abgelegt werden:

| Secret Key            | Beschreibung                          |
|-----------------------|---------------------------------------|
| AWS_ENDPOINTS         | Die URL des Backup-Speicher           |
| AWS_ACCESS_KEY_ID     | Die ID des AccessKey für Longhorn     |
| AWS_SECRET_ACCESS_KEY | Das Secret zum AccessKey für Longhorn |

Das Secret kann beispielsweise mit diesem Befehl angelegt werden:

```shell
kubectl create secret generic longhorn-backup-target --namespace=longhorn-system \
--from-literal=AWS_ENDPOINTS=http://192.168.56.1:9000 \
--from-literal=AWS_ACCESS_KEY_ID=MY-ACCESS-KEY \
--from-literal=AWS_SECRET_ACCESS_KEY=MY-ACCESS-SECRET123
```

Das Secret muss im selben Kubernetes-Namespace wie `longhorn` angelegt werden.

### Longhorn konfigurieren

Die Helm-Values von Longhorn um folgende Werte erweitern:

```yaml
defaultBackupStore:
  backupTarget: s3://longhorn@dummyregion/
  backupTargetCredentialSecret: longhorn-backup-target
```
Für das Backup sind folgende Parameter in den Values relevant:

| Parameter                                         | Beschreibung                                                                                         |
|---------------------------------------------------|------------------------------------------------------------------------------------------------------|
| `defaultBackupStore.backupTarget`                 | Die Adresse des Speicherorts (Buckets) innerhalb des Backup-Speichers: `s3://<BUCKET_NAME>@<REGION>` |
| `defaultBackupStore.backupTargetCredentialSecret` | Der Name des oben erstellten Secrets, dass die Zugangsdaten zum Backup-Speicher enthält              |

### Snapshot-API

Falls das Kubernetes-Cluster nicht die Snapshot-API unterstützt muss ebenfalls ein Snapshot-Controller installiert werden.
Dies ist der Fall, wenn man zum Beispiel `k3s` als Kubernetes-Distribution verwendet.

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

## Velero

Velero benötigt zur Ablage der Backups ebenfalls Konfiguration.

### Secret für den Backup-Speicher erstellen

Velero-Backups werden ebenfalls im oben beschriebenen Backup-Speicher abgelegt. Dazu benötigt Velero Zugriff auf den Speicher.
Die dafür benötigten Parameter müssen in einem Kubernetes-Secret abgelegt werden:

| Secret Key            | Beschreibung                        |
|-----------------------|-------------------------------------|
| aws_access_key_id     | Die ID des AccessKey für Velero     |
| aws_secret_access_key | Das Secret zum AccessKey für Velero |

Das Secret für wird als Datei in Velero verwendet und muss daher nach folgendem Beispiel angelegt werden:

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

Das Secret muss im selben Kubernetes-Namespace wie `velero` angelegt werden.

### Velero konfigurieren

Mit dem Attribut `valuesYamlOverwrite` lassen sich auch hier beliebige Konfigurationen hinzufügen oder überschreiben:

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
Die folgenden Parameter in der `valuesYamlOverwrite` sind für die Backup-Konfiguration relevant:

| Parameter                                                              | Beschreibung                                                                                      |
|------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
| `velero.credentials.existingSecret`                                    | Der Name des oben erstellen Secrets, dass die Zugangsdaten für Velero zum Backup-Speicher enthält |
| `velero.configuration.backupStorageLocation[default].bucket`           | Der Name des Buckets für Velero innerhalb des Backup-Speichers                                    |
| `velero.configuration.backupStorageLocation[default].config.s3Url`     | Die URL des Backup-Speichers                                                                      |
| `velero.configuration.backupStorageLocation[default].config.publicUrl` | Die öffentliche URL des Backup-Speichers                                                          |

Die erstellte `yaml`-Datei für die Velero-Komponente kann mit folgendem Befehl angewendet werden:

```shell
kubectl --namespace ecosystem apply -f k8s-velero.yaml
```