# Backup Operator Email Dispatch

This document explains how the k8s-backup-operator component interacts with Prometheus, Grafana, Postfix and Mailpit(for local dev) to manage, monitor, and alert on backup/restore statuses.

## Architecture Overview
   Here is a diagram providing an overview of the process of sending alerts for the backup process.
   ![](img/cloudogu_backup_email.png)
## Component Workflows

### 1. k8s-backup-operator
* **Action**: Accepts request to trigger a backup.
* **Metrics Tracking**: Updates an internal Prometheus metric based on the outcome of the backup/restore
  * Success: (e.g., backup_status_transitions_total{name="backup-20260708-1624",namespace="ecosystem",to="completed"} 4)
  * Fail: (e.g., backup_status_transitions_total{name="backup-20260708-1624",namespace="ecosystem",to="failed"} 4).
* **Storage**: Stores metrics in memory locally.
* **Exposition**: Exposes these metrics on a `/metrics` HTTP endpoint on port 8080.

Note: In order to let prometheus scrape the data, we need to set the following value:

`metrics.serviceMonitor.enabled=true`
This ensures that there is a service monitor : k8s-backup-operator-servicemonitor.
Through this service monitor,  prometheus knows through which pod the data can be accessed.


### 2. Prometheus
* **Action**: Acts as the central time-series database.
* **Scraping**: Periodically pulls data from the pod's `/metrics` endpoint.
* **Storage**: Saves the scraped state for historical querying.

### 3. Grafana

* **Action**: Evaluates the backup metrics using an alert rule that is configured in grafana ([backupalerts.yaml](https://github.com/cloudogu/grafana/blob/develop/resources/default-provisioning/alerting/backupalerts.yaml)).
* **Schedule**: Runs every 10 minutes.
* **Logic**: Queries Prometheus. If the data has changed (indicating a new failure or success status), it triggers an alert instance.
* **Routing**: Forwards the alert notification to the configured SMTP contact point.

### 4. Mail Delivery (Postfix,  Mailpit)

#### Postfix
* **Role**: Production Mail Transfer Agent (MTA).
* **Action**: Grafana connects to Postfix via SMTP. Postfix routes and delivers the actual alert email to the recipient's external inbox.

#### Development: Mailpit
* **Role**: Local email testing tool.
* **Action**: Receives forwarded emails from Postfix, stores them safely in memory, and displays them in a local web dashboard for developer review.

## Testing Email Delivery for Backup & Restore

To test email delivery for backup and restore, Mailpit must be deployed in the cluster.

### Preparing the cluster

This guide assumes a cluster with backup configured, which means that the `k8s-backup-operator` and
`k8s-backup-operator-crd` are available and Velero is configured.

To enable Prometheus to collect metrics, the Prometheus ServiceMonitor must be enabled in the backup operator.
This can be done via `valuesYamlOverwrite` when installing the operator:

```yaml
  valuesYamlOverwrite: |
    metrics:
      serviceMonitor:
        enabled: true
```

The following components must also be installed:

- Prometheus as a data source
- Grafana

When working on a Coder cluster, the `dogu.name=grafana` label must also be added to Grafana's Dogu custom resource.

### Mailpit

Mailpit is used to test email delivery. Four resources need to be created for this purpose: a Deployment, a Service,
an Ingress, and a NetworkPolicy. The file [mailpit.yaml](k8s-resources/mailpit.yaml) includes all the necessary resources and can be applied
to the cluster. It is located in the `k8s-resources` directory. The current namespace needs to be `ecosystem`

Next, edit the Postfix ConfigMap and set `relayhost` to

`relayhost: "[mailpit.ecosystem.svc.cluster.local]:1025"`

Then set up port forwarding for port 8025 of the Mailpit pod. For example, in k9s this can be done with the
``Shift+F`` keyboard shortcut, or from the command line with

```shell
kubectl port-forward -n ecosystem pods/mailpit 8025:8025
```

The UI should now be available at <http://localhost:8025>.

### Testing email delivery

In Grafana, the notification policies can be viewed under Alerting/Contact points. They can be tested in the detail
view ("View"). If the test is successful, the emails will appear in the Mailpit UI.
