package leases

import (
	"context"
	"fmt"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultName          = "k8s-backup-operator-lease"
	HolderNameAnnotation = "k8s.cloudogu.com/backup-operator-lease-holder-name"
	HolderKindAnnotation = "k8s.cloudogu.com/lease-holder-kind"
)

type State int

const (
	StateAcquired State = iota
	StateWaiting
	StateInvalid
	StateChanged
)

type Result struct {
	State      State
	HolderName string
}

type inspectionResult struct {
	activeHolder client.Object
	invalid      bool
}

type resolutionResult struct {
	holder client.Object
}

type Resolver interface {
	Kind() string
	Get(ctx context.Context, namespace, name string) (client.Object, error)
	IsTerminal(holder client.Object) bool
}

type Manager struct {
	client    client.Client
	namespace string
	name      string
	resolvers map[string]Resolver
}

func NewManager(k8sClient client.Client, namespace, name string, resolvers ...Resolver) *Manager {
	registered := make(map[string]Resolver, len(resolvers))
	for _, resolver := range resolvers {
		registered[resolver.Kind()] = resolver
	}
	return &Manager{client: k8sClient, namespace: namespace, name: name, resolvers: registered}
}

func (m *Manager) Acquire(ctx context.Context, holder client.Object, kind string) (Result, error) {
	key := client.ObjectKey{Namespace: m.namespace, Name: m.name}
	lease := &coordinationv1.Lease{}
	err := m.client.Get(ctx, key, lease)
	if apierrors.IsNotFound(err) {
		lease = NewLease(m.namespace, m.name, holder, kind)
		if createErr := m.client.Create(ctx, lease); createErr != nil {
			if apierrors.IsAlreadyExists(createErr) {
				return Result{State: StateChanged}, nil
			}
			return Result{}, fmt.Errorf("failed to create lease %s/%s: %w", key.Namespace, key.Name, createErr)
		}
		return Result{State: StateChanged}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("failed to get lease %s/%s: %w", key.Namespace, key.Name, err)
	}

	if IsHolder(lease, holder, kind) {
		return Result{State: StateAcquired, HolderName: holder.GetName()}, nil
	}

	leaseKind := lease.Annotations[HolderKindAnnotation]
	if leaseKind != "" {
		if _, known := m.resolvers[leaseKind]; !known {
			return Result{State: StateWaiting, HolderName: lease.Annotations[HolderNameAnnotation]}, nil
		}
	}

	inspection, inspectErr := m.inspect(ctx, lease)
	if inspectErr != nil {
		return Result{}, inspectErr
	}
	if inspection.invalid {
		return Result{State: StateInvalid}, nil
	}
	if inspection.activeHolder != nil {
		return Result{State: StateWaiting, HolderName: inspection.activeHolder.GetName()}, nil
	}

	Claim(lease, holder, kind)
	if updateErr := m.client.Update(ctx, lease); updateErr != nil {
		if apierrors.IsConflict(updateErr) {
			return Result{State: StateChanged}, nil
		}
		return Result{}, fmt.Errorf("failed to take over stale lease %s/%s: %w", key.Namespace, key.Name, updateErr)
	}
	return Result{State: StateChanged}, nil
}

func (m *Manager) Release(ctx context.Context, holder client.Object, kind string) (bool, error) {
	lease := &coordinationv1.Lease{}
	key := client.ObjectKey{Namespace: m.namespace, Name: m.name}
	if err := m.client.Get(ctx, key, lease); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get lease %s/%s for release: %w", key.Namespace, key.Name, err)
	}
	if !IsHolder(lease, holder, kind) {
		return false, nil
	}
	preconditions := metav1.Preconditions{UID: &lease.UID, ResourceVersion: &lease.ResourceVersion}
	if err := m.client.Delete(ctx, lease, &client.DeleteOptions{Preconditions: &preconditions}); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to release lease %s/%s: %w", key.Namespace, key.Name, err)
	}
	return true, nil
}

func NewLease(namespace, name string, holder client.Object, kind string) *coordinationv1.Lease {
	lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	Claim(lease, holder, kind)
	return lease
}

func Claim(lease *coordinationv1.Lease, holder client.Object, kind string) {
	if lease.Annotations == nil {
		lease.Annotations = map[string]string{}
	}
	now := metav1.NowMicro()
	lease.Annotations[HolderNameAnnotation] = holder.GetName()
	lease.Annotations[HolderKindAnnotation] = kind
	lease.Spec.HolderIdentity = ptr.To(string(holder.GetUID()))
	lease.Spec.AcquireTime = &now
	lease.Spec.RenewTime = &now
	lease.Spec.LeaseTransitions = ptr.To(ptr.Deref(lease.Spec.LeaseTransitions, 0) + 1)
}

func IsHolder(lease *coordinationv1.Lease, holder client.Object, kind string) bool {
	return HolderUID(lease) == holder.GetUID() && lease.Annotations[HolderNameAnnotation] == holder.GetName() && lease.Annotations[HolderKindAnnotation] == kind
}

func HolderUID(lease *coordinationv1.Lease) types.UID {
	return types.UID(ptr.Deref(lease.Spec.HolderIdentity, ""))
}

// inspect determines whether a structurally valid lease still has an active holder.
func (m *Manager) inspect(ctx context.Context, lease *coordinationv1.Lease) (inspectionResult, error) {
	uid := HolderUID(lease)
	name := lease.Annotations[HolderNameAnnotation]
	kind := lease.Annotations[HolderKindAnnotation]

	// Claim always writes UID, name, and kind together. Missing identity data therefore indicates a
	// malformed or externally modified lease and must not be repaired heuristically.
	if uid == "" || name == "" || kind == "" {
		return inspectionResult{invalid: true}, nil
	}

	resolver, ok := m.resolvers[kind]
	if !ok {
		// Acquire handles known foreign workflow kinds before inspection. Reaching this branch means
		// that the lease kind cannot be interpreted safely by this manager.
		return inspectionResult{invalid: true}, nil
	}

	resolution, err := m.resolve(ctx, lease, resolver, name, uid)
	return inspectionResult{activeHolder: resolution.holder}, err
}

// resolve looks up the named holder and verifies that it is the non-terminal object identified by the lease UID.
func (m *Manager) resolve(ctx context.Context, lease *coordinationv1.Lease, resolver Resolver, name string, uid types.UID) (resolutionResult, error) {
	holder, err := resolver.Get(ctx, lease.Namespace, name)
	if apierrors.IsNotFound(err) {
		// A deleted holder makes the lease stale; this is an expected result rather than an error.
		return resolutionResult{}, nil
	}
	if err != nil {
		return resolutionResult{}, fmt.Errorf("failed to resolve holder of lease %s/%s: %w", lease.Namespace, lease.Name, err)
	}
	// Terminal holders no longer block another operation from acquiring the lease.
	if holder == nil || resolver.IsTerminal(holder) {
		return resolutionResult{}, nil
	}
	// A matching name with a different UID represents a replacement object, not the original holder.
	if holder.GetUID() != uid {
		return resolutionResult{}, nil
	}
	return resolutionResult{holder: holder}, nil
}
