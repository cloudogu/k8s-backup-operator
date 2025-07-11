# Lokale Testumgebung

Voraussetzung für die Installation ist ein laufendes [Multinode-EcoSystem](https://github.com/cloudogu/k8s-ecosystem).
Mit der Default-Konfiguration wird als Storage-Provider bisher nur Longhorn im Multinode-EcoSystem unterstützt.
Theoretisch sollten sich aber auch andere CSI-fähige Storage-Provider konfigurieren lassen.

Es wird ein S3-kompatibler Objektspeicher benötigt, um Backups speichern zu können.
In diesem Beispiel wird dafür MinIO auf dem Host ausgeführt:
```shell
../../samples/setup/run_local_minio.sh
```

Die Weboberfläche von MinIO (http://localhost:9090) ist erreichbar. Es kann sich mit `admin123:admin123` angemeldet 
werden

Für die Kommunikation mit dem Minio werden Secrets benötigt. Diese können wie folgt ins Cluster eingespielt werden:
```shell
../../samples/setup/create_backup_secrets.sh
```

Das folgende Blueprint bietet eine grundlegende Konfiguration des Backup-Stacks mit allen nötigen Komponenten:

```shell
kubectl apply -f ../../samples/setup/blueprint_configure_backup.yaml --namespace=ecosystem
```

Damit der `k8s-backup-operator` mit `k8s-longhorn` kommunizieren kann, müssen die Network Policies aus dem Namespace 
`longhorn-system` entfernt werden. Sonst ist es dem `k8s-backup-operator` nicht möglich den `admission-controller` 
von `k8s-longhorn` zu erreichen. 

Vor einem Backup überprüfen, ob die Backup Storage Location erreichbar ist:
```shell
kubectl get backupstoragelocation --namespace=ecosystem
```

Anschließend kann ein Backup und Restore durchgeführt werden:
```shell
kubectl apply -f ../../samples/backup.yaml --namespace=ecosystem
```

```shell
kubectl apply -f ../../samples/restore.yaml --namespace=ecosystem
```