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

### Storage-Provisioner

Falls im Cluster kein Storage-Provisioner existiert kann `longhorn` installiert und verwendet werden.
Mit dem Attribute `valuesYamlOverwrite` können für die Backups URL und Credentials zu einem S3-Storage konfiguriert werden.

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
    backup:
      target:
        secret:
          # aws_endpoint is just the server url to the s3 compatible storage.
          aws_endpoint: http://192.168.56.1:9001 # Insert your s3 url here. Ensure that the bucket `longhorn` exists in the Storage
          aws_access_key_id: abcd1234
          aws_secret_access_key: abcc1234
```

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
Diese beinhaltet den Access-Key, Secret-Key und die URL des S3-Storage.
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
        secretContents:
          cloud: |
            [default]
            aws_access_key_id=abcd1234
            aws_secret_access_key=abcc1234
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

Die Felder `s3Url` und `publicUrl` sind dementsprechend anzupassen.

`kubectl --namespace ecosystem apply -f k8s-velero.yaml`

### Installation Backup-Operator

Anschließend kann der Backup-Operator mit seinen CRDs installiert werden:

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

Konfiguration k8s-longhorn-values.yaml:

```yaml
backup:
  target:
    secret:
      # aws_endpoint is just the server url to the s3 compatible storage.
      aws_endpoint: http://192.168.56.1:9001 # Insert your s3 url here. Ensure that the bucket `longhorn` exists in the Storage
      aws_access_key_id: abcd1234
      aws_secret_access_key: abcc1234
```

Installation:

`helm install k8s-longhorn oci://registry.cloudogu.com/k8s/k8s-longhorn --version 1.5.1-3 -f k8s-longhorn-values.yaml --namespace longhorn-system --create-namespace`

### Snapshot-API

Installation:

`helm install k8s-snapshot-controller-crd oci://registry.cloudogu.com/k8s/k8s-snapshot-controller-crd --version 5.0.1-5 --namespace ecosystem --create-namespace`

`helm install k8s-snapshot-controller oci://registry.cloudogu.com/k8s/k8s-snapshot-controller --version 5.0.1-5 --namespace ecosystem`

### Velero

```yaml
velero:
  credentials:
    useSecret: true
    secretContents:
      cloud: |
        [default]
        aws_access_key_id=abcd1234
        aws_secret_access_key=abcc1234
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

Die Felder `s3Url` und `publicUrl` sind dementsprechend anzupassen.

`helm install k8s-velero oci://registry.cloudogu.com/k8s/k8s-velero --version 5.0.2-4 -f k8s-velero-values.yaml --namespace ecosystem`

### Installation Backup-Operator

Anschließend kann der Backup-Operator mit seinen CRDs installiert werden:

`helm install k8s-backup-operator-crd oci://registry.cloudogu.com/k8s/k8s-backup-operator-crd --version 0.9.0 --namespace ecosystem`

`helm install k8s-backup-operator oci://registry.cloudogu.com/k8s/k8s-backup-operator --version 0.9.0 --namespace ecosystem`

Sind alle Komponenten auf Status `RUNNING` kann geprüft werden, ob die BackupStorageLocation verfügbar und der S3-Storage von Velero erreichbar ist.

`kubectl --namespace ecosystem get backupstoragelocation default`

Anschließend kann ein regulärer Restore durchgeführt werden. Siehe [Durchführung Restore](restore_de.md).

### Helm-Registry Logout

`helm registry logout registry.cloudogu.com`
