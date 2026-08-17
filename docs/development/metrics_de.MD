### Metrik für Condition-Übergänge

Persistierte Statuswechsel einer Backup-Condition werden mit `backup_condition_transitions_total` erfasst. Die Labels sind:

| Label | Bedeutung |
|---|---|
| `namespace` | Namespace des Backups. |
| `name` | Name des Backups. |
| `condition` | Condition-Typ: `Prepared`, `Deleting`, `Canceled` oder `Succeeded`. |
| `from` | Vorheriger Status: `Unknown`, `True` oder `False`. |
| `to` | Neuer Status: `Unknown`, `True` oder `False`. |

Der `conditionsUpdater` bestimmt die Übergänge vor dem Patch, erhöht die Counter aber erst nach einem erfolgreichen Status-Write. Fehlgeschlagene Patches werden daher nicht gezählt. Das erstmalige Anlegen einer Condition sowie Änderungen nur an `Reason`, `Message` oder Transition-Zeit gelten nicht als Statusübergang.

Nach dem Lesen eines Backups initialisiert jeder Reconcile alle sechs möglichen Wechsel für alle vier Condition-Typen mit `Add(0)`. Dadurch stehen pro Backup 24 Zeitreihen zur Verfügung, auch wenn ein bestimmter Übergang noch nie aufgetreten ist. Wiederholte Initialisierungen verändern die Counter nicht.

Eine beispielhafte Abfrage erfolgreicher Übergänge lautet:

```promql
sum by (condition) (
  increase(backup_condition_transitions_total{to="True"}[1h])
)
```

`backup_reconcile_total` zählt weiterhin alle Backup-Reconciles. `backup_status_transitions_total` zählt die persistierten Wechsel des skalaren Legacy-Felds und bleibt von den Condition-Übergängen getrennt.
