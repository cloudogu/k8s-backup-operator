# Set these to the desired values
ARTIFACT_ID=k8s-backup-operator
VERSION=1.4.7
IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG?=1.24
MAKEFILES_VERSION=10.1.1
STAGE?=production
MOCKERY_VERSION=v2.53.3
MOCKERY_IGNORED=vendor,build,docs,generated

ADDITIONAL_CLEAN=dist-clean

include build/make/variables.mk
include build/make/self-update.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/digital-signature.mk
include build/make/mocks.mk

K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
CRD_BACKUP_SOURCE = ${HELM_CRD_SOURCE_DIR}/templates/k8s.cloudogu.com_backups.yaml
CRD_RESTORE_SOURCE = ${HELM_CRD_SOURCE_DIR}/templates/k8s.cloudogu.com_restores.yaml
CRD_SCHEDULE_SOURCE = ${HELM_CRD_SOURCE_DIR}/templates/k8s.cloudogu.com_backupschedules.yaml
PRE_COMPILE=generate-deepcopy
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo template-stage template-log-level template-image-pull-policy
CRD_POST_MANIFEST_TARGETS = crd-add-labels crd-add-backup-labels
CHECK_VAR_TARGETS=check-all-vars
IMAGE_IMPORT_TARGET=image-import

include build/make/k8s-controller.mk

.PHONY: build-boot
build-boot: helm-apply kill-operator-pod ## Builds a new version of the operator and deploys it into the K8s-EcoSystem.
.PHONY: crd-add-backup-labels
crd-add-backup-labels: $(BINARY_YQ)
	@echo "Adding backup label to CRDs..."
	@for file in ${HELM_CRD_SOURCE_DIR}/templates/*.yaml ; do \
		$(BINARY_YQ) -i e ".metadata.labels.\"k8s.cloudogu.com/part-of\" = \"backup\"" $${file} ;\
	done

.PHONY: helm-values-update-image-version
helm-values-update-image-version: $(BINARY_YQ)
	@echo "Updating the image version in source values.yaml to ${VERSION}..."
	@$(BINARY_YQ) -i e ".manager.image.tag = \"${VERSION}\"" ${K8S_COMPONENT_SOURCE_VALUES}

.PHONY: helm-values-replace-image-repo
helm-values-replace-image-repo: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
      		echo "Setting dev image repo in target values.yaml!" ;\
    		$(BINARY_YQ) -i e ".manager.image.registry=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\1/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    		$(BINARY_YQ) -i e ".manager.image.repository=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\2/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    	fi

.PHONY: template-stage
template-stage: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
  		echo "Setting STAGE env in deployment to ${STAGE}!" ;\
		$(BINARY_YQ) -i e ".manager.env.stage=\"${STAGE}\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
	fi

.PHONY: template-log-level
template-log-level: ${BINARY_YQ}
	@if [[ "${STAGE}" == "development" ]]; then \
      echo "Setting LOG_LEVEL env in deployment to ${LOG_LEVEL}!" ; \
      $(BINARY_YQ) -i e ".manager.env.logLevel=\"${LOG_LEVEL}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
    fi

.PHONY: template-image-pull-policy
template-image-pull-policy: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
          echo "Setting pull policy to always!" ; \
          $(BINARY_YQ) -i e ".manager.imagePullPolicy=\"Always\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
    fi

.PHONY: kill-operator-pod
kill-operator-pod:
	@echo "Restarting k8s-backup-operator!"
	@kubectl -n ${NAMESPACE} delete pods -l "app.kubernetes.io/name=$(ARTIFACT_ID)"

##@ Debug

.PHONY: print-debug-info
print-debug-info: ## Generates indo and the list of environment variables required to start the operator in debug mode.
	@echo "The target generates a list of env variables required to start the operator in debug mode. These can be pasted directly into the 'go build' run configuration in IntelliJ to run and debug the operator on-demand."
	@echo "STAGE=$(STAGE);LOG_LEVEL=$(LOG_LEVEL);KUBECONFIG=$(KUBECONFIG);NAMESPACE=$(NAMESPACE);DOGU_REGISTRY_ENDPOINT=$(DOGU_REGISTRY_ENDPOINT);DOGU_REGISTRY_USERNAME=$(DOGU_REGISTRY_USERNAME);DOGU_REGISTRY_PASSWORD=$(DOGU_REGISTRY_PASSWORD);DOCKER_REGISTRY={\"auths\":{\"$(docker_registry_server)\":{\"username\":\"$(docker_registry_username)\",\"password\":\"$(docker_registry_password)\",\"email\":\"ignore@me.com\",\"auth\":\"ignoreMe\"}}}"
