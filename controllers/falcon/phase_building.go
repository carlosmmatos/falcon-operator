package falcon

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (r *FalconConfigReconciler) phaseBuildingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Building")

	err := r.ensureDockercfg(ctx, instance.ObjectMeta.Namespace)
	if err != nil {
		return r.error(ctx, instance, "Cannot find dockercfg secret from the current namespace", err)
	}
	imageStream, err := r.imageStream(ctx, instance.ObjectMeta.Namespace)
	if err != nil {
		return r.error(ctx, instance, "Cannot access image stream", err)
	}

	image := falcon_container.NewImageRefresher(ctx, logger, instance.Spec.FalconAPI.ApiConfig())
	err = image.Refresh(imageStream.Status.DockerImageRepository)
	if err != nil {
		return r.error(ctx, instance, "Cannot refresh Falcon Container Image", err)
	}
	logger.Info("Falcon Container Image pushed successfully")

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseConfiguring

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}

func (r *FalconConfigReconciler) ensureDockercfg(ctx context.Context, namespace string) error {
	dockercfg, err := r.getDockercfg(ctx, namespace)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/tmp/.dockercfg", dockercfg, 0600)
}

func (r *FalconConfigReconciler) getDockercfg(ctx context.Context, namespace string) ([]byte, error) {
	secrets := &corev1.SecretList{}
	err := r.Client.List(ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return []byte{}, err
	}

	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if secret.Type != "kubernetes.io/dockercfg" {
			continue
		}

		if secret.ObjectMeta.Annotations == nil || secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"] != "builder" {
			continue
		}

		value, ok := secret.Data[".dockercfg"]
		if !ok {
			continue
		}
		return value, nil
	}

	return []byte{}, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
}

func (r *FalconConfigReconciler) error(ctx context.Context, instance *falconv1alpha1.FalconConfig, message string, err error) (ctrl.Result, error) {
	userError := fmt.Errorf("%s %w", message, err)

	instance.Status.ErrorMessage = userError.Error()
	instance.Status.Phase = falconv1alpha1.PhaseDone
	_ = r.Client.Status().Update(ctx, instance)

	return ctrl.Result{}, userError

}