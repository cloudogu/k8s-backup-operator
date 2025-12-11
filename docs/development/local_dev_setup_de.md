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

Die Erweiterung von `ecosystem-core` in `../../samples/setup/additionalValues.yaml` bietet eine grundlegende Konfiguration des Backup-Stacks mit allen nötigen Komponenten.
Sie kann in einem Testcluster folgendermaßen eingespielt werden:

```shell
helm get values ecosystem-core -o yaml -n ecosystem > original.yaml

if yq --version 2>&1 | grep -qi "mikefarah"; then
  # mikefarah version of yq installed
  yq eval-all 'select(fi==0) * select(fi==1)' original.yaml ../../samples/setup/additionalValues.yaml > merge.yaml
else
  # kislyuk version of yq installed
  yq -y --sort-keys '. *= input' original.yaml ../../samples/setup/additionalValues.yaml > merge.yaml
fi

helm upgrade ecosystem-core oci://registry.cloudogu.com/k8s/ecosystem-core --version 2.0.2 -n ecosystem -f merge.yaml
```
Außerdem muss Longhorn korrekt konfiguriert werden:

```shell
helm get values longhorn -o yaml -n longhorn-system > longhorn_original.yaml

if yq --version 2>&1 | grep -qi "mikefarah"; then
  # mikefarah version of yq installed
  yq eval-all 'select(fi==0) * select(fi==1)' longhorn_original.yaml ../../samples/setup/longhornAdditionalValues.yaml > longhorn_merge.yaml
else
  # kislyuk version of yq installed
  yq -y --sort-keys '. *= input' longhorn_original.yaml ../../samples/setup/longhornAdditionalValues.yaml > longhorn_merge.yaml
fi

helm upgrade longhorn longhorn/longhorn --version 1.10.0 -n longhorn-system -f longhorn_merge.yaml
```

Damit der `k8s-backup-operator` mit `k8s-longhorn` kommunizieren kann, müssen die Network Policies aus dem Namespace 
`longhorn-system` entfernt werden, sofern vorhanden. Sonst ist es dem `k8s-backup-operator` nicht möglich
den `admission-controller `von `k8s-longhorn` zu erreichen.

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
