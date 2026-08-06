package velero

type Service interface {
	Update(namespace string, name string)
}
