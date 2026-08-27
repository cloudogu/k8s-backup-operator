# Testen des Email-Verands beim Backup & Restore

Um die Email-Zustellung beim Backup & Restore zu testen muss der Cluster mit einer Mailpit-Instanz aufgesetzt werden. 

## Cluster vorbereiten

Wir gehen hier von einem Cluster mit konfiguriertem Backup aus, das heißt der k8s-backup-operator sowie die 
k8s-backup-operator-crd sind verfügbar und velero ist konfiguriert.

Um das Sammeln der Metrics vom Backup-Operator zu aktivieren muss der Backup-Operator mit

```yaml
  valuesYamlOverwrite: |
    metrics:
      serviceMonitor:
        enabled: true
```

konfiguriert sein.

Zusätzlich müssen nun installiert werden:
- Loki
- Prometheus
- Grafana

Wird auf einem Coder-Cluster gearbeitet, muss bei der Dogu-CR von Grafana noch das Lebel ``dogu.name=grafana`` 
ergänzt werden

## Mailpit 

Zum Testen der Emails wird Mailpit verwendet. Dafür werden drei Resourcen angelegt: ein Deployment, ein Service und 
eine NetworkPolicy. Die dafür benötigte yaml-Datei befinden sich im Verzeichnis k8s-resources. 

Danach die ConfigMap von postfix editieren und den Wert ``relayhost`` auf

``relayhost: "[mailpit.ecosystem.svc.cluster.local]:1025"``

setzen.

Nun für den Mailpit-Pod einen Port-Forward für den Port 8ß25 einrichten (zum Beispiel unter k9s mit ``Shift+F``), 
um die UI unter localhost:6025 erreichen zu können.

## Mailversand testen

In Grafana können unter Meldungen/Kontaktpunkte die Benachrichtigungsrichtlinien angesehen werden. In der Detailansicht 
("Anzeigen") kann man diese testen. Ist der Test erfolgreich, werden in der UI die Mails  angezeigt.
