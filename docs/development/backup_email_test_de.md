# Testen des E-Mail-Versands für Backup & Restore

Um die E-Mail-Zustellung für Backup und Restore zu testen, muss Mailpit im Cluster bereitgestellt werden.

## Cluster vorbereiten

Wir gehen hier von einem Cluster mit konfiguriertem Backup aus, das heißt der `k8s-backup-operator` sowie die
`k8s-backup-operator-crd` sind verfügbar und Velero ist konfiguriert.

Um das Sammeln der Metriken durch Prometheus zu aktivieren, muss der ServiceMonitor für Prometheus im
Backup-Operator aktiviert werden. Dies geht über die valuesYamlOverwrite beim Installieren des Operators mit:

```yaml
  valuesYamlOverwrite: |
    metrics:
      serviceMonitor:
        enabled: true
```

Zusätzlich müssen folgende Komponenten installiert werden:

- Prometheus als Datenquelle
- Grafana

Wird auf einem Coder-Cluster gearbeitet, muss bei der Dogu-CR von Grafana noch das Label `dogu.name=grafana`
ergänzt werden.

## Mailpit

Zum Testen der E-Mails wird Mailpit verwendet. Dafür müssen vier Ressourcen angelegt werden: ein Deployment, ein Service,
ein Ingress und eine NetworkPolicy. Die Datei [mailpit.yaml](k8s-resources/mailpit.yaml) beinhaltet alles Notwendige und kann im Cluster
angewendet werden. Sie befindet sich im Verzeichnis `k8s-resources`. Der Namespace muss dabei auf `ecosystem` gesetzt 
sein.

Danach die ConfigMap von Postfix editieren und den Wert `relayhost` auf

`relayhost: "[mailpit.ecosystem.svc.cluster.local]:1025"`

setzen.

Nun für den Mailpit-Pod einen Port-Forward für den Port 8025 einrichten. Dies geht zum Beispiel unter k9s mit
der Tastenkombination ``Shift+F`` oder über die Kommandozeile mit

```shell
kubectl port-forward -n ecosystem pods/mailpit 8025:8025
```

Nun sollte die UI unter <http://localhost:8025> erreicht werden können.

## Mailversand testen

In Grafana können unter Meldungen/Kontaktpunkte die Benachrichtigungsrichtlinien angesehen werden. In der Detailansicht
("Anzeigen") kann man diese testen. Ist der Test erfolgreich, werden in der UI von Mailpit die E-Mails angezeigt.
