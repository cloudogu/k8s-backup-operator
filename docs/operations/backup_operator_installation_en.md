# Installing the backup operator

The backup operator can then be installed with its Component-CRs:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator-crd
spec:
  name: k8s-backup-operator-crd
  namespace: k8s
  version: <0.0.0>
```

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-backup-operator
spec:
  name: k8s-backup-operator
  namespace: k8s
  version: <0.0.0>
```

`kubectl --namespace ecosystem apply -f k8s-backup-operator-crd.yaml`

`kubectl --namespace ecosystem apply -f k8s-backup-operator.yaml`

