# Installation des Backup-Operators

Voraussetzung für die Installation ist ein laufendes [Multinode-EcoSystem][mn-ecosystem-repo].
Mit der Default-Konfiguration wird als Storage-Provider bisher nur Longhorn im Multinode-EcoSystem unterstützt.
Theoretisch sollten sich aber auch andere CSI-fähige Storage-Provider konfigurieren lassen.

[mn-ecosystem-repo]: https://github.com/cloudogu/k8s-ecosystem

Es wird MinIO auf dem Host benötigt, um Backups speichern zu können:
```shell
docker run -d --name minio \
-p 9000:9000 -p 9090:9090 \
-e "MINIO_ROOT_USER=MINIOADMIN" \
-e "MINIO_ROOT_PASSWORD=MINIOADMINPW" \
quay.io/minio/minio \
server /data --console-address ":9090"
```
Dort in der Weboberfläche (http://localhost:9090) zwei Buckets `velero` und `longhorn`
und einen Access Key `longhorn-test-key` mit dem Secret Key `longhorn-test-secret-key` anlegen.
(Longhorn und Velero sind schon entsprechend vorkonfiguriert, müssen also nicht angepasst werden.)

Des Weiteren müssen [k8s-snapshot-controller][snapshot-ctrl-repo] und [k8s-velero][velero-repo] als Komponenten installiert werden.
Dazu die Repositories auschecken und darin folgende Befehle ausführen:
```shell
# nur im snapshot-controller:
make crd-component-apply
# für snapshot-controller und velero:
make component-apply
```

[snapshot-ctrl-repo]: https://github.com/cloudogu/k8s-snapshot-controller
[velero-repo]: https://github.com/cloudogu/k8s-velero

Auch der [k8s-backup-operator][backup-op-repo] kann mit unseren Makefiles installiert werden:
```shell
make crd-component-apply component-apply
```

[backup-op-repo]: https://github.com/cloudogu/k8s-backup-operator