package schedule

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dlclark/regexp2"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
)

// imageTagValidator defines a regexp string that validates a container reference. These include:
//   - standard DNS rules
//   - optional hostnames
//   - optional port numbers like :30099
//   - optional tags
var imageTagValidationString = "^(?:(?=[^:\\/]{1,253})(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(?:\\.(?!-)[a-zA-Z0-9-]{1,63}(?<!-))*(?::[0-9]{1,5})?/)?((?![._-])(?:[a-z0-9._-]*)(?<![._-])(?:/(?![._-])[a-z0-9._-]*(?<![._-]))*)(?::(?![.-])[a-zA-Z0-9_.-]{1,128})?$"
var imageTagValidationRegexp, _ = regexp2.Compile(imageTagValidationString, regexp2.None)

type operatorImageGetter struct {
	configmapClient kubernetes.Interface
	namespace       string
}

// NewOperatorImageGetter creates an image getter that reads from the operator's additional-images ConfigMap.
func NewOperatorImageGetter(client kubernetes.Interface, namespace string) OperatorImageGetter {
	return &operatorImageGetter{configmapClient: client, namespace: namespace}
}

// ImageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
func (oig *operatorImageGetter) ImageForKey(ctx context.Context, key string) (string, error) {
	logging.Info(ctx, "reading backup operator image to use from ConfigMap", "configMap", config.OperatorAdditionalImagesConfigmapName, "key", key)

	configMap, err := oig.configmapClient.CoreV1().
		ConfigMaps(oig.namespace).
		Get(ctx, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error while getting configmap '%s': %w", config.OperatorAdditionalImagesConfigmapName, err)
	}

	imageTag := configMap.Data[key]
	if imageTag == "" {
		return "", fmt.Errorf("image %q in configmap %q is empty", key, config.OperatorAdditionalImagesConfigmapName)
	}

	err = verifyImageTag(imageTag)
	if err != nil {
		return "", fmt.Errorf("configmap '%s' contains an invalid image tag: %w", config.OperatorAdditionalImagesConfigmapName, err)
	}

	logging.Info(ctx, "read backup operator image to use from ConfigMap", "configMap", config.OperatorAdditionalImagesConfigmapName, "key", key, "image", imageTag)
	return imageTag, nil
}

// verifyImageTag checks whether imageTag is a valid container image reference.
func verifyImageTag(imageTag string) error {
	matched, err := imageTagValidationRegexp.MatchString(imageTag)
	if err != nil {
		return fmt.Errorf("image tag validation of %s failed: %w", imageTag, err)
	}
	if !matched {
		return fmt.Errorf("image tag '%s' seems invalid (please compare it with the image tag specs)", imageTag)
	}
	return nil
}
