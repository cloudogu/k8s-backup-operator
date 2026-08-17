# Metriken

Der Operator stellt Counter für Reconciles sowie für persistierte Status- und Condition-Übergänge von Backups und Restores bereit.

| Metrik | Labels | Bedeutung |
|---|---|---|
| `backup_reconcile_total` | – | Anzahl aller Backup-Reconciles. |
| `restore_reconcile_total` | – | Anzahl aller Restore-Reconciles. |
| `backup_status_transitions_total` | `namespace`, `name`, `to` | Persistierte Wechsel des veralteten skalaren Backup-Status. |
| `restore_status_transitions_total` | `namespace`, `name`, `to`, `backup_name` | Persistierte Wechsel des veralteten skalaren Restore-Status. |
| `backup_condition_transitions_total` | `namespace`, `name`, `condition`, `from`, `to` | Persistierte Statuswechsel einer Backup-Condition. |
| `restore_condition_transitions_total` | `namespace`, `name`, `backup_name`, `condition`, `from`, `to` | Persistierte Statuswechsel einer Restore-Condition. |

## Condition-Übergänge

Die Condition-Metriken zählen ausschließlich tatsächliche Wechsel zwischen `Unknown`, `True` und `False`, beispielsweise `Prepared: Unknown -> True`. Das erstmalige Anlegen einer Condition sowie Änderungen nur an `Reason`, `Message`, `ObservedGeneration` oder der Transition-Zeit werden nicht gezählt.

Der jeweilige Condition-Updater erhöht den Counter erst nach einem erfolgreichen Status-Write. Fehlgeschlagene Writes und verworfene Conflict-Versuche erscheinen daher nicht in der Metrik. Bei einem Conflict werden die Übergänge gegen den neu gelesenen Ressourcenstatus erneut bestimmt.

Nach dem Lesen einer Ressource initialisiert jeder Reconcile alle sechs möglichen gerichteten Statuswechsel mit `Add(0)`. Dadurch sind auch noch nie beobachtete Übergänge als Null-Zeitreihen verfügbar:

| Ressource | Conditions | Zeitreihen pro Ressource |
|---|---|---|
| Backup | `Prepared`, `Deleting`, `Canceled`, `Succeeded` | 4 × 6 = 24 |
| Restore | `Successful`, `Prepared`, `ProviderRestoreSuccessful`, `WorkloadsRecovered`, `BackupsSynchronized` | 5 × 6 = 30 |

Beispiel für erfolgreich erreichte Meilensteine:

```promql
sum by (condition) (
  increase(backup_condition_transitions_total{from="Unknown", to="True"}[1h])
)
```

Für Restores wird entsprechend `restore_condition_transitions_total` verwendet. Mit dem zusätzlichen Label `backup_name` kann nach dem wiederhergestellten Backup gruppiert oder gefiltert werden.

## Legacy-Status und Reconciles

Die Metriken `backup_status_transitions_total` und `restore_status_transitions_total` bleiben für den skalaren Kompatibilitätsstatus `status.status` bestehen. Neue Auswertungen sollten bevorzugt die detaillierteren Condition-Übergänge verwenden.

`backup_reconcile_total` und `restore_reconcile_total` zählen jeden Aufruf des jeweiligen Reconcilers unabhängig vom Ergebnis. Wie alle Prometheus-Counter beginnen die Werte nach einem Neustart des Operators erneut bei null.
