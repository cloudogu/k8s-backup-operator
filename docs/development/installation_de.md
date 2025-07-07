# Installation des Backup-Operators

Voraussetzung für die Installation ist ein laufendes [Multinode-EcoSystem][mn-ecosystem-repo].
Mit der Default-Konfiguration wird als Storage-Provider bisher nur Longhorn im Multinode-EcoSystem unterstützt.
Theoretisch sollten sich aber auch andere CSI-fähige Storage-Provider konfigurieren lassen.

[mn-ecosystem-repo]: https://github.com/cloudogu/k8s-ecosystem

Es wird MinIO auf dem Host benötigt, um Backups speichern zu können:
```shell
../../samples/setup/run_local_minio.sh
```

Für die Kommunikation mit dem Minio werden Secrets benötigt. Diese können wie folgt ins Cluster eingespielt werden:
```shell
../../samples/setup/create_backup_secrets.sh
```



In der Weboberfläche von MinIO (http://localhost:9090) kann sich mit den Zugangsdaten `admin123:admin123` angemeldet 
werden. Anschließend zwei Buckets `velero` und `longhorn` erstellen. Zusätzlich werden zwei Access Keys benötigt:
- Name: `MY-ACCESS-KEY` Secret: `MY-ACCESS-SECRET123`
- Name: `MY-VELERO-ACCESS-KEY` Secret: `MY-VELERO.ACCESS-SECRET123`
Longhorn und Velero sind schon entsprechend vorkonfiguriert, müssen also nicht angepasst werden.


Das folgende Blueprint bietet eine grundlegende Konfiguration des Backup-Stacks mit allen nötigen Komponenten:

```shell
kubectl apply -f ../../samples/setup/blueprint_configure_backup.yaml --namespace=ecosystem
```

Vor einem Backup überprüfen, ob die Backup Storage Location erreichbar ist:
```shell
kubectl get backupstoragelocation --namespace=ecosystem
```

Anschließend kann ein Backup durchgeführt werden:
```shell
kubectl apply -f ../../samples/backup.yaml --namespace=ecosystem
```