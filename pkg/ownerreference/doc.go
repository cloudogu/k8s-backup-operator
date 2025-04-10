// Package ownerreference contains logic to backup and restore owner references of k8s.cloudogu.com objects and its
// immediate children.
//
// Basically, this logic only exists because the k8s-dogu-operator fails to properly reconcile owner references to its
// resources after a restore action. This package and its logic can be dropped if the k8s-dogu-operator supports proper
// owner reference reconciliation.
package ownerreference
