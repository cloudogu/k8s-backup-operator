# Backup creation

To back up the Cloudogu EcoSystem one has to apply a backup custom resource:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Backup
metadata:
  name: backup-sample
spec:
  provider: velero # only velero and "" (defaults to velero) is supported.
```