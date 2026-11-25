# Restores ausführen

Eine Ressource analog zu diesem Beispiel muss im Namespace des `k8s-backup-operator` angelegt werden:
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Restore
metadata:
  name: restore-sample
spec:
  provider: velero # aktuell wird nur velero unterstützt
  backupName: backup-sample # der Name des Backups das eingespielt werden soll
```

Bevor die Wiederherstellung ausgeführt wird,
werden Ressourcen in diesem Namensraum, die für den Sicherungsprozess irrelevant sind, entfernt, um einen sauberen Zustand herzustellen.
Dies ist besonders notwendig
da installierte Dogus, die nicht in der Sicherung enthalten sind, nicht mehr funktionieren würden, weil die Sicherung ihre Datenbank nicht enthält.

## Globale Konfiguration der Zielinstanz behalten

Bei einem Restore aus dem Backup einer anderen Dogu-v2-Instanz kann mit `spec.options.ignoreGlobalConfig: true`
die globale Konfiguration der Zielinstanz behalten werden. Der Operator erhält die vorhandene globale ConfigMap beim Aufräumen,
auch wenn sie das Label `k8s.cloudogu.com/backup-scope` trägt. Velero überspringt die globale ConfigMap aus dem Backup
auch dann, wenn im Ziel keine globale ConfigMap vorhanden ist.

Die Option setzt `k8s-backup-operator-crd` ab Version `1.10.0` voraus.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Restore
metadata:
  name: restore-preserve-global-config
spec:
  provider: velero
  backupName: backup-sample
  options:
    ignoreGlobalConfig: true
```

Fehlt die Option oder ist sie `false`, wird die globale Konfiguration wie bisher aus dem Backup wiederhergestellt.
Die Option gilt für den jeweiligen Restore. Die globale Konfiguration bleibt Teil des Backups.

## Backup nach einem Restore starten

Ein während dieses Restores erstelltes Backup wartet und startet erst, nachdem der Restore-Vorgang abgeschlossen ist.
Dies garantiert jedoch nicht, dass das gesamte Cloudogu EcoSystem und alle PVCs bereit sind. Ein unmittelbar danach
gestartetes Backup kann daher teilweise fehlschlagen. Unter
[Backup unmittelbar nach einem Restore starten](./backup_de.md#backup-unmittelbar-nach-einem-restore-starten) sind das
empfohlene Vorgehen und die Begründung für dieses Verhalten beschrieben.
