# Installation des Backup-Operators

Der Backup-Operator lässt sich gewöhnlich in einem bestehenden Cloudogu EcoSystem oder leeren Cluster installieren.

## Installation mit bestehenden Cloudogu EcoSystem

In einem bestehenden Cloudogu EcoSystem wird der Backup-Operator über den Component-Operator installiert.
Dafür muss eine Custom Resource `Component` für den Backup-Operator und seine eigenen CRDs angelegt werden.

### Abhängigkeiten

Vorher sollten aber die Abhängigkeiten des Operators installiert werden. Der Backup-Operator benötigt einen Backup-Provider.
Aktuell wird `velero` als Provider unterstützt.
Ist in dem Cluster keine Snapshot-API verfügbar muss ebenfalls ein Snapshot-Controller installiert werden.
Das Gleiche gilt für den Storage-Provisioner.

#### Backup-Speicher

Die Speicherung der Backups erfolgt in einem S3-kompatiblen Objektspeicher, z.B. [Minio](https://min.io/).
Dieser Speicher sollte sich außerhalb des Kubernetes Clusters befinden, damit bei einem Ausfall des Clusters die Backups weiterhin vorhanden und sicher sind.
Daher muss die Installation und der Betrieb des Backup-Speichers separat vom CES durchgeführt werden.

### Storage-Provisioner

Falls im Cluster kein Storage-Provisioner existiert kann `longhorn` installiert und verwendet werden.

#### Secret für den Backup-Speicher erstellen

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
--from-literal=AWS_ENDPOINTS=https://192.168.56.1:9000 \
--from-literal=AWS_ACCESS_KEY_ID=MY-ACCESS-KEY \
--from-literal=AWS_SECRET_ACCESS_KEY=MY-ACCESS-SECRET123
```

Das Secret muss im selben Kubernetes-Namespace wie `longhorn` angelegt werden.

#### Longhorn konfigurieren

Mit dem Attribut `valuesYamlOverwrite` können für die Backups URL und Credentials zu dem Backup-Speicher konfiguriert werden.

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

Für das Backup sind folgende Parameter in der `valuesYamlOverwrite` relevant:

| Parameter                                               | Beschreibung                                                                                         |
|---------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| `longhorn.defaultSettings.backupTarget`                 | Die Adresse des Speicherorts (Buckets) innerhalb des Backup-Speichers: `s3://<BUCKET_NAME>@<REGION>` |
| `longhorn.defaultSettings.backupTargetCredentialSecret` | Der Name des oben erstellten Secrets, dass die Zugangsdaten zum Backup-Speicher enthält               |

Die erstellte `yaml`-Datei für die Longhorn-Komponente kann mit folgendem Befehl angewendet werden:

`kubectl --namespace ecosystem apply -f k8s-longhorn.yaml`

#### Snapshot-API

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

#### Velero

Velero benötigt zur Ablage der Backups ebenfalls Konfiguration.

#### Secret für den Backup-Speicher erstellen

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

#### Velero konfigurieren

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

### Installation Backup-Operator

Anschließend kann der Backup-Operator mit seinen Komponenten-CRs installiert werden:

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
> Die Versionen der Komponenten können über das Attribut `version` angepasst passt werden:

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


Sind alle Komponenten auf Status `RUNNING` kann geprüft werden, ob die BackupStorageLocation verfügbar und der S3-Storage von Velero erreichbar ist.

`kubectl --namespace ecosystem get backupstoragelocation default`

Anschließend kann ein reguläres Backup durchgeführt werden. Siehe [Durchführung Backup](backup_de.md).

## Installation in einem leeren Cluster

Es kann durchaus Sinn machen Backups nicht in einem bestehenden Cloudogu EcoSystem wiederherzustellen.
Dies ist sinnvoll, wenn man zum Beispiel ein Restore in einen neuen Cluster ausführen möchte.
Dadurch erspart man sich ein initiales Setup des Cloudogu EcoSystems.

Der Unterschied zu der Methode [Installation in einem bestehenden Cluster](#installation-mit-bestehenden-cloudogu-ecosystem) ist, dass hier die Installation nicht mit dem Komponenten-Operator durchgeführt werden können.
Stattdessen werden die regulären Helm-Charts angewendet. Konfigurationen liegen nicht in der Component-CR, sondern in einer values.yaml.

Die Bestimmung der Abhängigkeiten bleibt bei dieser Methode gleich:
- Falls kein Storage-Provisioner existiert, muss `k8s-longhorn` installiert werden.
- Fall keine Snapshot-API existiert, muss `k8s-snapshot-controller-crd` und `k8s-snapshot-controller` installiert werden.
- Als Backup-Provider muss `k8s-velero` installiert werden.

### Helm-Registry Login

Da in einem bestehenden Cluster der Komponenten-Operator Credentials für die Helm-Registry hat, muss man bei dieser Methode sich direkt mit der Registry authentifizieren.

`helm registry login registry.cloudogu.com`

### Storage-Provisioner

Erstellung des Longhorn-Secrets für den Backup-Speicher:

```shell
kubectl create secret generic longhorn-backup-target --namespace=longhorn-system \
--from-literal=AWS_ENDPOINTS=http://192.168.56.1:9000 \
--from-literal=AWS_ACCESS_KEY_ID=MY-ACCESS-KEY \
--from-literal=AWS_SECRET_ACCESS_KEY=MY-ACCESS-SECRET123
```

Konfiguration k8s-longhorn-values.yaml:

```yaml
longhorn:
  defaultSettings:
    backupTarget: s3://longhorn@dummyregion/
    backupTargetCredentialSecret: long-backup-target
```

Installation:

`helm install k8s-longhorn oci://registry.cloudogu.com/k8s/k8s-longhorn --version 1.5.1-3 -f k8s-longhorn-values.yaml --namespace longhorn-system --create-namespace`

### Snapshot-API

Installation:

`helm install k8s-snapshot-controller-crd oci://registry.cloudogu.com/k8s/k8s-snapshot-controller-crd --version 5.0.1-5 --namespace ecosystem --create-namespace`

`helm install k8s-snapshot-controller oci://registry.cloudogu.com/k8s/k8s-snapshot-controller --version 5.0.1-5 --namespace ecosystem`

### Velero

Erstellung des Velero-Secrets für den Backup-Speicher:

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

Konfiguration values.yaml:

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

`helm install k8s-velero oci://registry.cloudogu.com/k8s/k8s-velero --version 5.0.2-4 -f values.yaml --namespace ecosystem`

### Installation Backup-Operator

Anschließend kann der Backup-Operator installiert werden:

`helm install k8s-backup-operator-crd oci://registry.cloudogu.com/k8s/k8s-backup-operator-crd --version 0.9.0 --namespace ecosystem`

`helm install k8s-backup-operator oci://registry.cloudogu.com/k8s/k8s-backup-operator --version 0.9.0 --namespace ecosystem`

Sind alle Komponenten auf Status `RUNNING` kann geprüft werden, ob die BackupStorageLocation verfügbar und der S3-Storage von Velero erreichbar ist.

`kubectl --namespace ecosystem get backupstoragelocation default`

Anschließend kann ein regulärer Restore durchgeführt werden. Siehe [Durchführung Restore](restore_de.md).

### Helm-Registry Logout

`helm registry logout registry.cloudogu.com`
