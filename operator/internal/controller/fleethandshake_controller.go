package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fleetv1alpha1api "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	handshakev1alpha1 "github.com/rptcloud/fleet-handshake/operator/api/v1alpha1"
)

type FleetHandshakeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=riverpointtechnology.com,resources=fleethandshakes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=riverpointtechnology.com,resources=fleethandshakes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=riverpointtechnology.com,resources=fleethandshakes/finalizers,verbs=update
//+kubebuilder:rbac:groups=fleet.cattle.io,resources=bundles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *FleetHandshakeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling FleetHandshake", "request", req)

	var fleetHandshake handshakev1alpha1.FleetHandshake
	if err := r.Get(ctx, req.NamespacedName, &fleetHandshake); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Unable to fetch FleetHandshake")
		return ctrl.Result{}, err
	}

	var secret corev1.Secret
	secretKey := types.NamespacedName{
		Name:      fleetHandshake.Spec.SecretName,
		Namespace: fleetHandshake.Spec.SecretNamespace,
	}
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf(`{"message": "secret not found", "name": %q, "namespace": %q}`, fleetHandshake.Spec.SecretName, fleetHandshake.Spec.SecretNamespace))
			fleetHandshake.Status.Status = "Missing"
			r.Status().Update(ctx, &fleetHandshake)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Unable to fetch Secret")
		return ctrl.Result{}, err
	}

	jsonSecret, err := json.Marshal(&corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: fleetHandshake.Spec.TargetNamespace,
		},
		Data:       secret.Data,
		StringData: secret.StringData,
		Type:       secret.Type,
	})
	if err != nil {
		logger.Error(err, "Unable to marshal Secret to JSON")
		fleetHandshake.Status.Status = "Error"
		r.Status().Update(ctx, &fleetHandshake)
		return ctrl.Result{}, err
	}

	bundle := &fleetv1alpha1api.Bundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetHandshake.Name,
			Namespace: fleetHandshake.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: fleetHandshake.APIVersion,
				Kind:       fleetHandshake.Kind,
				Name:       fleetHandshake.Name,
				UID:        fleetHandshake.UID,
			}},
		},
		Spec: fleetv1alpha1api.BundleSpec{
			Resources: []fleetv1alpha1api.BundleResource{
				{
					Name:    fmt.Sprintf("%s.json", secret.Name),
					Content: string(jsonSecret),
				},
			},
			Targets: fleetHandshake.Spec.Targets,
		},
	}

	// Check if Bundle exists
	existingBundle := &fleetv1alpha1api.Bundle{}
	err = r.Get(ctx, types.NamespacedName{Name: bundle.Name, Namespace: bundle.Namespace}, existingBundle)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new Bundle
			logger.Info(fmt.Sprintf(`{"message": "creating new Bundle", "name": %q, "namespace": %q}`, bundle.Name, bundle.Namespace))
			if err := r.Create(ctx, bundle); err != nil {
				logger.Error(err, "Unable to create Bundle")
				fleetHandshake.Status.Status = "Error"
				r.Status().Update(ctx, &fleetHandshake)
				return ctrl.Result{}, err
			}
		} else {
			logger.Error(err, "Unable to fetch existing Bundle")
			fleetHandshake.Status.Status = "Error"
			r.Status().Update(ctx, &fleetHandshake)
			return ctrl.Result{}, err
		}
	} else {
		// Update existing Bundle if content has changed
		if !reflect.DeepEqual(existingBundle.Spec, bundle.Spec) {
			existingBundle.Spec = bundle.Spec
			logger.Info(fmt.Sprintf(`{"message": "updating existing Bundle", "name": %q, "namespace": %q}`, bundle.Name, bundle.Namespace))
			if err := r.Update(ctx, existingBundle); err != nil {
				logger.Error(err, "Unable to update Bundle")
				fleetHandshake.Status.Status = "Error"
				r.Status().Update(ctx, &fleetHandshake)
				return ctrl.Result{}, err
			}
		} else {
			logger.Info(fmt.Sprintf(`{"message": "Bundle content unchanged", "name": %q, "namespace": %q}`, bundle.Name, bundle.Namespace))
		}
	}

	fleetHandshake.Status.Status = "Synced"
	if err := r.Status().Update(ctx, &fleetHandshake); err != nil {
		logger.Error(err, "Unable to update FleetHandshake status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FleetHandshakeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&handshakev1alpha1.FleetHandshake{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findFleetHandshakesForSecret),
		).
		Complete(r)
}

func (r *FleetHandshakeReconciler) findFleetHandshakesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return []reconcile.Request{}
	}

	fleetHandshakes := &handshakev1alpha1.FleetHandshakeList{}
	err := r.List(ctx, fleetHandshakes)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0)
	logger := log.FromContext(ctx)
	for _, fh := range fleetHandshakes.Items {
		if fh.Spec.SecretName == secret.Name && fh.Spec.SecretNamespace == secret.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      fh.Name,
					Namespace: fh.Namespace,
				},
			})
			logger.Info(fmt.Sprintf(`{"message": "tracking FleetHandshake for updated secret", "fleethandshake": %q, "secret": %q, "namespace": %q}`, fh.Name, fh.Spec.SecretName, fh.Spec.SecretNamespace))
		}
	}

	return requests
}
