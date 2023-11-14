# Automated Backup Deletion and Retention

Backups can be automatically garbage-collected.
By default, no backups are deleted.

To control which backups get deleted, one of several retention strategies can be enabled.
Garbage-collection and retention can be configured through the values of this component:

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
      # The strategy defining which backups get deleted. By default, it is keepAll.
      strategy: keepLastSevenDays
      # The cron pattern defining how often backups are garbage-collected.
      # By default this is "0 * * * *", so every hour.
      garbageCollectionCron: "0 */3 * * *"
```

The following strategies are available:
- `keepAll` - No backups get automatically deleted. This is the default.
- `removeAllButKeepLatest` - Only the latest Backup is retained. All other backups get automatically deleted.
- `keepLastSevenDays` - All backups from the last seven days are retained. Backups older than that get deleted.
- `keep7Days1Month1Quarter1Year` - Keeps all backups from the last seven days and the oldest of the last month, quarter, half year and year.
  The following table shows its behavior:
  
  | retained backups |  time period    |
  |------------------|-----------------|
  | ALL              |  0 - 7 days     |
  | 1                |  8 - 30 days    |
  | 1                |  31 - 90 days   |
  | 1                |  91 - 180 days  |
  | 1                |  181 - 360 days |
  However, it must be considered that e.g., the oldest backup of the last year is not always 360 days old
  but rather its age varies between 181 and 360 days.
