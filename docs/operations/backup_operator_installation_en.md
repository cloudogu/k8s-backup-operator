# Installing the backup operator

The backup operator can then be installed with its Component-CRs:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator-crd
spec:
  name: k8s-backup-operator-crd
  namespace: k8s
  version: <0.0.0>
```

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator
spec:
  name: k8s-backup-operator
  namespace: k8s
  version: <0.0.0>
```

`kubectl --namespace ecosystem apply -f k8s-backup-operator-crd.yaml`

`kubectl --namespace ecosystem apply -f k8s-backup-operator.yaml`

## Configuration keys in values.yaml

| Konfigurationsschlüssel         | Beschreibung                                                                                                                                                                                                                                                                                                                               | Standardwert               |
|---------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| global/imagePullSecrets/name    | Name of the secret used to retrieve images from the registry                                                                                                                                                                                                                                                                               | "ces-container-registries" |
| global/networkPolicies/enabled  | Enable network rules                                                                                                                                                                                                                                                                                                                       | true                       |
| retention/strategy              | Strategy for retaining backups. <br/> *keepAll*: all backups are retained<br/>*removeAllButKeepLatest*: all except the latest are removed<br/>*keepLastSevenDays*: all from the last 7 days are retained<br/>*keep7Days1Month1Quarter1Year*: retains the oldest backup from 7 days, one month, one quarter, and one year ago, respectively | "keepAll"                  |
| retention/garbageCollectionCron | Time at which backups are deleted, in CRON notation                                                                                                                                                                                                                                                                                        | 0 * * * *                  |
| manager/env/logLevel            | Log level of the pod                                                                                                                                                                                                                                                                                                                       | info                       |
| initContainer/schedule/name     | Name of the backup schedule for automatically performed backups                                                                                                                                                                                                                                                                            | ces-schedule               |
| initContainer/schedule/cron     | Schedule for automatic backups in CRON format                                                                                                                                                                                                                                                                                              | 00 02 * * *                |
| metrics/serviceMonitor/enabled  | Activates the service monitor to send metrics. Required for sending emails via Grafana.                                                                                                                                                                                                                                                    | true                       |
