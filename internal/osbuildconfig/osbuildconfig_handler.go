// using --old-style-config flag in order to separate the generated files for each type (types, server, and client files)
//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package=restapi -old-config-style -generate=types -o ../../restapi/osbuildconfig_webhook_trigger_types.go  ../../osbuildconfig_webhook_trigger.yaml
//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package=restapi -old-config-style -generate=chi-server -o ../../restapi/osbuildconfig_webhook_trigger_server.go  ../../osbuildconfig_webhook_trigger.yaml
//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package=restapi -old-config-style -generate=client -o ../../restapi/osbuildconfig_webhook_trigger_client.go  ../../osbuildconfig_webhook_trigger.yaml

package osbuildconfig

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/project-flotta/osbuild-operator/internal/httpapi"
	loggerutil "github.com/project-flotta/osbuild-operator/internal/logger"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	repositoryosbuildconfig "github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/restapi"
)

type OSBuildConfigHandler struct {
	OSBuildConfigRepository repositoryosbuildconfig.Repository
	SecretRepository        secret.Repository
	OSBuildCRCreator        manifests.OSBuildCRCreator
}

func NewOSBuildConfigHandler(osBuildConfigRepository repositoryosbuildconfig.Repository,
	secretRepository secret.Repository, osBuildCRCreator manifests.OSBuildCRCreator) *OSBuildConfigHandler {
	return &OSBuildConfigHandler{
		OSBuildConfigRepository: osBuildConfigRepository,
		SecretRepository:        secretRepository,
		OSBuildCRCreator:        osBuildCRCreator,
	}
}
func (o *OSBuildConfigHandler) TriggerBuild(w http.ResponseWriter, r *http.Request, namespace string, name string, params restapi.TriggerBuildParams) {
	logger, err := loggerutil.Logger(httpapi.GlobalHttpAPIConf.LogLevel)
	if err != nil {
		return
	}

	logger.Info("New OSBuild trigger was sent ", "OSBuildConfig ", name, " namespace ", namespace)

	osBuildConfig, err := o.OSBuildConfigRepository.Read(r.Context(), name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error("resource OSBuildConfig not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error(err, fmt.Sprintf("cannot retrieve OSBuildConfig %s", name))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if osBuildConfig.Spec.Triggers.WebHook == nil {
		logger.Error("resource OSBuildConfig doesn't support triggers by webhook")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	secretName := osBuildConfig.Spec.Triggers.WebHook.SecretReference.Name
	secretVal := params.Secret
	webhookSecret, err := o.SecretRepository.Read(r.Context(), secretName, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error("secret not found", "secret", secretName)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error(err, "cannot read secret")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if string(webhookSecret.Data["WebHookSecretKey"]) != secretVal {
		logger.Error("secret value is forbidden")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = o.OSBuildCRCreator.Create(r.Context(), osBuildConfig)
	if err != nil {
		logger.Error(err, "cannot create new OSBuild CR")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Info("new CR of OSBuild was created")
	w.WriteHeader(http.StatusOK)
}
