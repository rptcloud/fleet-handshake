package controller_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	handshakev1alpha1 "github.com/rptcloud/fleet-handshake/operator/api/v1alpha1"
)

var _ = Describe("FleetHandshake Controller", func() {
	const (
		FleetHandshakeName      = "test-fleethandshake"
		FleetHandshakeNamespace = "default"
		SecretName              = "test-secret"
		SecretNamespace         = "default"
		TargetNamespace         = "target-namespace"

		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating a FleetHandshake", func() {
		It("Should create a Bundle", func() {

			By("Creating the Secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: SecretNamespace,
				},
				Data: map[string][]byte{
					"test-key": []byte("test-value"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("Creating a new FleetHandshake")
			ctx := context.Background()
			fleetHandshake := &handshakev1alpha1.FleetHandshake{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "handshake.fleet.io/v1alpha1",
					Kind:       "FleetHandshake",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      FleetHandshakeName,
					Namespace: FleetHandshakeNamespace,
				},
				Spec: handshakev1alpha1.FleetHandshakeSpec{
					SecretName:      SecretName,
					SecretNamespace: SecretNamespace,
					TargetNamespace: TargetNamespace,
					Targets: []fleetv1alpha1.BundleTarget{
						{
							ClusterName: "test-cluster",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, fleetHandshake)).Should(Succeed())

			By("Checking if the Bundle is created")
			bundleKey := types.NamespacedName{
				Name:      FleetHandshakeName,
				Namespace: FleetHandshakeNamespace,
			}
			createdBundle := &fleetv1alpha1.Bundle{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, bundleKey, createdBundle)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdBundle.Spec.Targets).To(HaveLen(1))
			Expect(createdBundle.Spec.Targets[0].ClusterName).To(Equal("test-cluster"))

			By("Checking if the FleetHandshake status is updated")
			updatedFleetHandshake := &handshakev1alpha1.FleetHandshake{}
			Eventually(func() string {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: FleetHandshakeName, Namespace: FleetHandshakeNamespace}, updatedFleetHandshake)
				if err != nil {
					return ""
				}
				return updatedFleetHandshake.Status.Status
			}, timeout, interval).Should(Equal("Synced"))
		})
	})

	Context("When creating a FleetHandshake with non-existent Secret", func() {
		It("Should set the FleetHandshake status to Missing", func() {
			ctx := context.Background()
			fleetHandshake := &handshakev1alpha1.FleetHandshake{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "handshake.fleet.io/v1alpha1",
					Kind:       "FleetHandshake",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-secret-fleethandshake",
					Namespace: FleetHandshakeNamespace,
				},
				Spec: handshakev1alpha1.FleetHandshakeSpec{
					SecretName:      "non-existent-secret",
					SecretNamespace: SecretNamespace,
					TargetNamespace: TargetNamespace,
					Targets: []fleetv1alpha1.BundleTarget{
						{
							ClusterName: "test-cluster",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, fleetHandshake)).Should(Succeed())

			By("Checking if the FleetHandshake status is set to Missing")
			Eventually(func() string {
				updatedFleetHandshake := &handshakev1alpha1.FleetHandshake{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "missing-secret-fleethandshake", Namespace: FleetHandshakeNamespace}, updatedFleetHandshake)
				if err != nil {
					return ""
				}
				return updatedFleetHandshake.Status.Status
			}, timeout, interval).Should(Equal("Missing"))
		})
	})

	Context("When updating a Secret referenced by a FleetHandshake", func() {
		It("Should update the corresponding Bundle", func() {
			ctx := context.Background()

			By("Creating a new FleetHandshake and Secret")
			fleetHandshake := &handshakev1alpha1.FleetHandshake{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "riverpointtechnology.com/v1alpha1",
					Kind:       "FleetHandshake",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "update-test-fleethandshake",
					Namespace: FleetHandshakeNamespace,
				},
				Spec: handshakev1alpha1.FleetHandshakeSpec{
					SecretName:      "update-test-secret",
					SecretNamespace: SecretNamespace,
					TargetNamespace: TargetNamespace,
					Targets: []fleetv1alpha1.BundleTarget{
						{
							ClusterName: "test-cluster",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, fleetHandshake)).Should(Succeed())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "update-test-secret",
					Namespace: SecretNamespace,
				},
				Data: map[string][]byte{
					"test-key": []byte("initial-value"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("Waiting for the initial Bundle to be created")
			bundleKey := types.NamespacedName{
				Name:      fleetHandshake.Name,
				Namespace: fleetHandshake.Namespace,
			}
			createdBundle := &fleetv1alpha1.Bundle{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, bundleKey, createdBundle)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the Secret")
			updatedSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "update-test-secret", Namespace: SecretNamespace}, updatedSecret)).Should(Succeed())
			updatedSecret.Data["test-key"] = []byte("updated-value")
			Expect(k8sClient.Update(ctx, updatedSecret)).Should(Succeed())

			By("Checking if the Secret is updated")
			Eventually(func() string {
				var sec corev1.Secret
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "update-test-secret", Namespace: SecretNamespace}, &sec)
				if err != nil {
					return ""
				}
				return string(sec.Data["test-key"])
			}, timeout, interval).Should(ContainSubstring("updated-value"))

			By("Checking if the FleetHandshake status is still Synced")
			Eventually(func() string {
				var handshake handshakev1alpha1.FleetHandshake
				err := k8sClient.Get(ctx, bundleKey, &handshake)
				if err != nil {
					return ""
				}
				return handshake.Status.Status
			}, timeout, interval).Should(Equal("Synced"))

			By("Checking if the Bundle is updated")
			Eventually(func() string {
				var bundle fleetv1alpha1.Bundle
				err := k8sClient.Get(ctx, bundleKey, &bundle)
				if err != nil {
					return ""
				}
				var s *corev1.Secret
				err = json.Unmarshal([]byte(bundle.Spec.Resources[0].Content), &s)
				if err != nil {
					return ""
				}
				return string(s.Data["test-key"])
			}, timeout, interval).Should(ContainSubstring("updated-value"))
		})
	})
})
