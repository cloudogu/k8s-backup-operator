# Installation des Backup-Operators

Den Backup-Operator mit seinen Komponenten-CRs installieren:

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

## Konfiguration über die values.yaml

| Konfigurationsschlüssel         | Beschreibung                                                                                                                                                                                                                                                                                                                                                  | Standardwert               |
|---------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| global/imagePullSecrets/name    | Name des Secrets um Images aus der Registry zu ziehen                                                                                                                                                                                                                                                                                                         | "ces-container-registries" |
| global/networkPolicies/enabled  | Netzwerkregeln aktivieren                                                                                                                                                                                                                                                                                                                                     | true                       |
| retention/strategy              | Strategie, wie Backups behalten werden.<br/> *keepAll*: alle Backups werden behalten<br/>*removeAllButKeepLatest*: alle bis auf das letzte werden entfernt<br/>*keepLastSevenDays*: alle aus den letzten 7 Tagen werden behalten<br/>*keep7Days1Month1Quarter1Year*: behält jeweils das älteste Backup von vor 7 Tage, einem Monat, einem Quartal, einem Jahr | "keepAll"                  |
| retention/garbageCollectionCron | Zeitpunkt, an dem die Backups gelöscht werden, in CRON Notation                                                                                                                                                                                                                                                                                               | 0 * * * *                  |
| manager/env/logLevel            | Loglevel des Pods                                                                                                                                                                                                                                                                                                                                             | info                       |
| initContainer/schedule/name     | Name des Backup-Zeitplans für automatisch durchgeführte Backups                                                                                                                                                                                                                                                                                               | ces-schedule               |
| initContainer/schedule/cron     | Zeitplan für automatische Backups in CRON-Format                                                                                                                                                                                                                                                                                                              | 00 02 * * *                |
| metrics/serviceMonitor/enabled  | Aktiviert den Servicemonitor, um Metriken zu senden. Wird für das Versenden von Mails über Grafana benötigt.                                                                                                                                                                                                                                                  | true                       |
