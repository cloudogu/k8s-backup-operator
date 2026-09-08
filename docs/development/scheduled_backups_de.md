# Geplante Backups

Geplante Backups werden durch das Anlegen einer  [`BackupSchedule`-Ressource](../operations/scheduled_backups_de.md) erstellt. 
Der Backup-Operator überführt jeden erstellten `BackupSchedule` in einen Kubernetes-`CronJob` namens
`backup-schedule-<Name-der-BackupSchedule>` im selben Namespace.

## Reconciliation

Der Controller überwacht sowohl `BackupSchedule`-Ressourcen als auch die zugehörigen `CronJob`-Ressourcen.
Während der Reconciliation führt er folgende Schritte aus:

1. Er ergänzt Labels und einen Finalizer an dem `BackupSchedule`.
2. Er prüft, ob `spec.schedule` einen standardkonformen Cron-Ausdruck enthält.
3. Ist dies der Fall, wird der zugehörige `CronJob` erstellt oder aktualisiert, sodass Zeitplan, Labels, Owner-Referenz, Pod-Template,
   Operator-Image und Image-Pull-Secrets dem gewünschten Zustand entsprechen.
4. Er hält das Ergebnis in den Status-Conditions `Accepted` und `Ready` des `BackupSchedule` fest.

Änderungen am zugehörigen `CronJob` oder dessen Löschung lösen eine weitere Reconciliation aus,
sodass der Controller den gewünschten Zustand wiederherstellt.

## Erstellen eines Backups

Zu jedem geplanten Zeitpunkt startet der `CronJob` das Image des Backup-Operators mit dem Unterbefehl
`scheduled-backup`. Dieser Prozess erstellt eine [`Backup`-Ressource](../operations/backup_de.md) im selben Namespace
und übernimmt den in der `BackupSchedule` konfigurierten Provider.
Der Name des Backups besteht aus dem Namen des `BackupSchedule` und dem Erstellungszeitstempel, zum Beispiel
`daily-2026-08-19t02.00.00`.

## Löschung

Beim Löschen eines `BackupSchedule` fordert der Controller die Löschung des verwalteten `CronJob` an und entfernt 
anschließend den Finalizer am `BackupSchedule`. Ein bereits fehlender `CronJob` gilt als gelöscht 
und wird nicht weiter behandelt. Schlägt das Löschen des `CronJob` oder das Entfernen des Finalizers fehl, bleibt der
Finalizer erhalten und die Reconciliation wird erneut versucht.

## Kubernetes-Events

Der Controller zeichnet die folgenden Events an der `BackupSchedule` auf:

| Typ | Reason | Bedeutung |
|-----|--------|-----------|
| Normal | `CronJobCreated` | Der verwaltete `CronJob` wurde erstellt. |
| Normal | `CronJobUpdated` | Der verwaltete `CronJob` wurde aktualisiert. |
| Normal | `CronJobDeletionRequested` | Die Löschung des verwalteten `CronJob` wurde angefordert. |
| Warning | `InvalidSchedule` | `spec.schedule` ist ungültig. |
| Warning | `CronJobSynchronizationFailed` | Das Erstellen oder Aktualisieren des verwalteten `CronJob` ist fehlgeschlagen. |
| Warning | `CronJobDeletionFailed` | Das Löschen des verwalteten `CronJob` ist fehlgeschlagen. |
| Warning | `FinalizerRemovalFailed` | Das Entfernen des `BackupSchedule`-Finalizers ist fehlgeschlagen. |
