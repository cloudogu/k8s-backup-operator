# Automatisierte Löschung und Aufbewahrung von Backups

Backups können automatisch gelöscht werden.
Standardmäßig werden keine Backups gelöscht.

Um zu steuern, welche Backups gelöscht werden, kann eine von mehreren Aufbewahrungsstrategien aktiviert werden.
Automatisches Löschen und die Aufbewahrung können über die Values der Komponente konfiguriert werden.:

```yaml backup-operator-component.yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator
spec:
  name: k8s-backup-operator
  namespace: k8s
  deployNamespace: ecosystem
  valuesYamlOverwrite: |
    retention:
      # Die Strategie mit welcher Backups gelöscht werden. Standardmäßig ist es keepAll.
      strategy: keepLastSevenDays
      # Dieses Cron-Pattern definiert, wie oft Backups gelöscht werden sollen.
      # Standardeinstellung ist "0 * * * *", also jede volle Stunde.
      garbageCollectionCron: "0 */3 * * *"
```

Die folgenden Strategien sind verfügbar:
- `keepAll` - Keine Backups werden automatisch gelöscht. Dies ist die Standardeinstellung.
- `removeAllButKeepLatest` - Nur das letzte Backup wird beibehalten. Alle anderen Backups werden automatisch gelöscht.
- `keepLastSevenDays` - Alle Backups der letzten sieben Tage werden beibehalten. Backups, die älter sind, werden gelöscht.
- `keep7Days1Month1Quarter1Year` - Behält alle Backups der letzten sieben Tage und das älteste des letzten Monats, Quartals, halben Jahres und Jahres.
  Die folgende Tabelle zeigt das Verhalten:
  
  | aufbewahrte Backups | Zeitraum       |
  |---------------------|----------------|
  | ALL                 | 0 - 7 Tage     |
  | 1                   | 8 - 30 Tage    |
  | 1                   | 31 - 90 Tage   |
  | 1                   | 91 - 180 Tage  |
  | 1                   | 181 - 360 Tage |
  Allerdings ist zu beachten, dass z.B. das älteste Backup des letzten Jahres nicht immer 360 Tage alt ist,
  sondern dessen Alter zwischen 181 und 360 Tagen schwankt.