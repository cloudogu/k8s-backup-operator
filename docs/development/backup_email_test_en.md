# Testing Email Delivery for Backup & Restore

To test email delivery for backup and restore, Mailpit must be deployed in the cluster.

## Preparing the cluster

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

## Mailpit

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

## Testing email delivery

In Grafana, the notification policies can be viewed under Alerting/Contact points. They can be tested in the detail
view ("View"). If the test is successful, the emails will appear in the Mailpit UI.
