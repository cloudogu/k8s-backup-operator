# Installation of the backup operator

Prerequisite for the installation is a running [Multinode EcoSystem][mn-ecosystem-repo].
With the default configuration, only Longhorn is supported as a storage provider in the Multinode EcoSystem.
Theoretically, however, it should also be possible to configure other CSI-capable storage providers.

[mn-ecosystem-repo]: https://github.com/cloudogu/k8s-ecosystem

MinIO is required on the host to store backups:
```shell
docker run -d --name minio \
-p 9000:9000 -p 9090:9090 \
-e "MINIO_ROOT_USER=MINIOADMIN" \
-e "MINIO_ROOT_PASSWORD=MINIOADMINPW" \
quay.io/minio/minio \
server /data --console-address ":9090"
```
In the web interface (http://localhost:9090) two buckets `velero` and `longhorn`
and an access key `longhorn-test-key` with the secret key `longhorn-test-secret-key` must be configured.
(Longhorn and Velero are already preconfigured accordingly, so they do not need to be adjusted).

Furthermore, [k8s-snapshot-controller][snapshot-ctrl-repo] and [k8s-velero][velero-repo] have to be installed as components.
To do this, check out the repositories and execute the following commands inside:
```shell
# only in the snapshot-controller:
make crd-component-apply
# for snapshot-controller and velero:
make component-apply
```

[snapshot-ctrl-repo]: https://github.com/cloudogu/k8s-snapshot-controller
[velero-repo]: https://github.com/cloudogu/k8s-velero

The [k8s-backup-operator][backup-op-repo] can be installed using the makefiles as well:
```shell
make crd-component-apply component-apply
```

[backup-op-repo]: https://github.com/cloudogu/k8s-backup-operator