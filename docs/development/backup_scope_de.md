# Backup-Umfang

Aktuell wird das Backup einzelner Dogus noch nicht unterstützt.
Derzeit werden die Daten aller Dogus und die globale Konfiguration gesichert.
Konkret werden Ressourcen mit dem Label-Schlüssel `dogu.name` sowie dem Label `k8s.cloudogu.com/type: global-config` ausgewählt.
Gesichert werden nur ConfigMaps, Secrets, PersistentVolumeClaims und die Dogu-Ressource selbst.
Da die Dogu-Ressource gesichert wird, kann der Dogu-Operator alle anderen Ressourcen, die nicht im Backup enthalten sind, erneut erzeugen.

Zusätzliche Ressourcen können in das Backup aufgenommen werden, indem ihnen das Label `k8s.cloudogu.com/backup-scope` gesetzt wird, zum Beispiel auf `PersistentVolumeClaims` von Komponenten.
Dabei gilt weiterhin die Beschränkung auf die zuvor genannten Ressourcentypen.

## Restore-Verhalten für zusätzliche Ressourcen

Während eines Restores löscht der Operator zunächst alle Ressourcen, die zum Restore-Umfang gehören, und erstellt sie danach aus dem Backup neu.
Das gilt sowohl für Dogus als auch für zusätzliche Ressourcen, die über `k8s.cloudogu.com/backup-scope` ausgewählt wurden.

Wenn ein Workload eine solche zusätzliche Ressource mountet oder anderweitig verwendet, muss er vor dem Restore herunter- und nach dem Restore wieder hochskaliert werden.
Andernfalls können Pods weiter auf Ressourcen zugreifen, die gerade gelöscht und neu erstellt werden.

Der Restore-Ablauf ist:

1. Das System wird in den Maintenance Mode versetzt.
2. Für den Restore markierte Workloads werden herunterskaliert.
3. Dogus und zusätzliche Ressourcen aus dem Restore-Umfang werden gelöscht.
4. Der Restore wird beim konfigurierten Provider ausgelöst.
5. Die zuvor herunterskalierten Workloads werden wieder hochskaliert.

## Labels

Die folgenden Labels werden zusammen verwendet:

### `k8s.cloudogu.com/backup-scope`

Dieses Label wird auf zusätzliche Ressourcen gesetzt, die zum Backup- und Restore-Umfang gehören sollen.

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
