# Backup-Umfang

## Backup-Units

Ein Cloudogu-Backup enthält verschiedene Backup-Units, die in der Backup-CR angegeben bzw. ausgeschlossen werden.
Welche Ressourcen für welche Units gesichert werden, wird in den folgenden Abschnitten beschrieben.

### Global-Config

Hierbei wird nur die ConfigMap mit dem Label `k8s.cloudogu.com/type: global-config` gesichert.

### V2-Dogus

V2-Dogus können nur alle zusammen in einer Unit gesichert werden.
Gibt es im Cluster keine V2-Dogus, wird diese Unit nicht erzeugt.

Hierbei werden von allen V2-Dogus alle ConfigMaps, Secrets, PersistentVolumeClaims und die Dogu-Ressource selbst
gesichert, insofern sie dem Label `dogu.name` ODER dem Label `k8s.cloudogu.com/dogu.name` entsprechen.
Der Wert des Labels muss hierbei dem Dogu-Namen entsprechen.
Die Labels müssen nur an der Dogu-Ressource explizit gesetzt werden,
was durch einen ValidatingAdmissionWebhook erzwungen wird.
An allen anderen Ressourcen werden die Labels automatisch vom Dogu-Operator gesetzt.

### Dogus

Hiermit sind aktuell V3-Dogus gemeint. Diese werden einzeln gesichert und können somit auch einzeln ausgeschlossen werden.

Hierbei werden je Dogu ConfigMaps, Secrets, PersistentVolumeClaims und die Dogu-Ressource selbst gesichert,
insofern der Wert des Labels `k8s.cloudogu.com/dogu.name` mit dem Dogu-Namen übereinstimmt.

### Komponenten

Komponenten werden alle zusammen in einer Unit gesichert.
Gibt es im Cluster keine Komponenten mit Ressourcen, für die ein Backup nötig ist, wird diese Unit nicht erzeugt.

Hierbei werden von allen Komponenten alle ConfigMaps, Secrets, PersistentVolumeClaims und ServiceAccountProducer
gesichert, insofern sie das Backup-Scope-Label `k8s.cloudogu.com/backup-scope` besitzen UND der Wert des
`k8s.cloudogu.com/component.name`-Labels mit einem der im Backup nicht ausgeschlossenen und im Cluster vorhandenen
Komponenten-Namen übereinstimmt.

#### Komponenten als Service-Account-Producer

Der Component-Operator setzt das Label `k8s.cloudogu.com/backup-scope` an jeden `ServiceAccountProducer` einer Component.
Der Cleanup vor einem Restore löscht dann den Producer, und der Service-Account-Operator löscht die Secrets aller
`ServiceAccountRequests` dieses Producers.
Stellt der Restore den Producer wieder her, rotiert der Service-Account-Operator den Account und legt die Secrets mit
neuen Credentials an, sodass sie wieder zu den neuen Daten passen.

## Restore-Verhalten für Component Ressourcen

Während eines Restores löscht der Operator zunächst alle Ressourcen, die zum Restore-Umfang gehören, und erstellt sie danach aus dem Backup neu.
Das gilt sowohl für Dogus als auch für Component-Ressourcen, die über `k8s.cloudogu.com/backup-scope` ausgewählt wurden.
Bei einem Restore aus Backup-Units gehören nur die wiederhergestellten Dogus und Components zum Restore-Umfang:
Component-Ressourcen werden nur gelöscht, wenn ihr Label `k8s.cloudogu.com/component.name` eine wiederhergestellte Component nennt.
Per `excludedDogus`, `excludedComponents` oder `options.ignoreV2Dogus` ausgeschlossene Dogus und Components bleiben unverändert.
Bei einem Restore aus einem Legacy-Backup werden weiterhin alle Dogus und alle Ressourcen mit `k8s.cloudogu.com/backup-scope` gelöscht.

Wenn ein Workload eine solche zusätzliche Ressource mountet oder anderweitig verwendet, muss er vor dem Restore herunter- und nach dem Restore wieder hochskaliert werden.
Dafür muss der Workload mit `k8s.cloudogu.com/restore-scaledown-scope` gelabelt sein.
Andernfalls können Pods weiter auf Ressourcen zugreifen, die gerade gelöscht und neu erstellt werden.

Der Restore-Ablauf ist:

1. Das System wird in den Maintenance Mode versetzt.
2. Für den Restore markierte Workloads werden herunterskaliert.
3. Dogus (inklusive ihrer PVCs) und Component-Ressourcen aus dem Restore-Umfang werden gelöscht.
4. Der Restore wird beim konfigurierten Provider ausgelöst.
5. Die zuvor herunterskalierten Workloads werden wieder hochskaliert.

## Labels

Die folgenden Labels werden zusammen verwendet:

### `k8s.cloudogu.com/backup-scope`

Dieses Label wird auf Component-Ressourcen gesetzt, die zum Backup- und Restore-Umfang gehören sollen.

Beispiel:

```yaml
metadata:
  labels:
    k8s.cloudogu.com/backup-scope: component-a
```

### `k8s.cloudogu.com/restore-scaledown-scope`

Dieses Label wird auf Workloads gesetzt, die Ressourcen mit `k8s.cloudogu.com/backup-scope` mounten oder anderweitig davon abhängen.
Der Operator prüft derzeit nur, ob dieses Label vorhanden ist. Der konkrete Label-Wert wird beim Herunter- und Hochskalieren nicht ausgewertet.

Beispiel:

```yaml
metadata:
  labels:
    k8s.cloudogu.com/restore-scaledown-scope: component-a
```

Das bedeutet:

- Ressourcen mit `k8s.cloudogu.com/backup-scope: component-a` werden in diesem Scope gelöscht und wiederhergestellt.
- Workloads mit `k8s.cloudogu.com/restore-scaledown-scope` werden vor dem Restore herunterskaliert und danach wieder hochskaliert.

In der Praxis kann derselbe Wert weiterhin zur Dokumentation und zur besseren Nachvollziehbarkeit verwendet werden, er wird vom Backup-Operator aktuell aber nicht interpretiert.

### `k8s.cloudogu.com/restore-scaledown-replicas`

Dieses Label wird während des Restores vom Backup-Operator verwaltet.
Beim Herunterskalieren speichert der Operator darin die ursprüngliche Replica-Anzahl eines Workloads und verwendet diesen Wert anschließend, um den vorherigen Zustand wiederherzustellen.

Dieses Label sollte nicht manuell gesetzt oder gepflegt werden.

## Beispiel

Wenn eine Komponente ein PVC verwendet, das gesichert und wiederhergestellt werden soll:

1. Das PVC mit `k8s.cloudogu.com/backup-scope: component-a` labeln.
2. Jeden Workload, der dieses PVC mountet, mit `k8s.cloudogu.com/restore-scaledown-scope: component-a` labeln.
3. `k8s.cloudogu.com/restore-scaledown-replicas` nicht selbst setzen; dieses Label wird während des Restores vom Operator geschrieben und wieder entfernt.

Im Beispiel wird derselbe Wert aus Gründen der Lesbarkeit verwendet. Die aktuelle Implementierung verlangt für `k8s.cloudogu.com/restore-scaledown-scope` jedoch nur die Existenz des Labels.
