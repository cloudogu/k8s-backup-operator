package velero

import (
	"fmt"
	"hash/fnv"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const (
	// maxObjectNameLength is the maximum length of metadata.name.
	maxObjectNameLength = 253
	// unitNameHashLength is the number of hex characters of the hash which makes truncated unit names unique.
	unitNameHashLength = 8
)

// UnitObjectName concatenates the parent name, the unit type and, for dogu units, the dogu name.
// Names exceeding the maximum object name length are truncated and suffixed with a hash of the full
// name, so that two long names which only differ at the end stay unique.
func UnitObjectName(parentName string, unit backupv1.UnitIdentifier) string {
	name := fmt.Sprintf("%s-%s", parentName, unit.Type)
	if unit.Name != "" {
		name = fmt.Sprintf("%s-%s", name, unit.Name)
	}

	if len(name) <= maxObjectNameLength {
		return name
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(name)) // hash.Hash.Write never returns an error
	suffix := fmt.Sprintf("%016x", hasher.Sum64())[:unitNameHashLength]

	return fmt.Sprintf("%s-%s", name[:maxObjectNameLength-unitNameHashLength-1], suffix)
}
