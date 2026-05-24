/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	storagev1alpha1 "github.com/a1010s/s3-monitor-operator/api/v1alpha1"
)

var _ = Describe("BucketMonitor Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName = "test-bucketmonitor"
			secretName   = "test-s3-creds"
			namespace    = "default"
		)

		ctx := context.Background()

		namespacedName := types.NamespacedName{Name: resourceName, Namespace: namespace}

		BeforeEach(func() {
			By("creating the credentials secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"AWS_ACCESS_KEY_ID":     []byte("test-key"),
					"AWS_SECRET_ACCESS_KEY": []byte("test-secret"),
				},
			}
			err := k8sClient.Create(ctx, secret)
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the BucketMonitor resource")
			bm := &storagev1alpha1.BucketMonitor{}
			err = k8sClient.Get(ctx, namespacedName, bm)
			if err != nil && errors.IsNotFound(err) {
				resource := &storagev1alpha1.BucketMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: storagev1alpha1.BucketMonitorSpec{
						Bucket:            "test-bucket",
						Endpoint:          "localhost:9000",
						CredentialsSecret: secretName,
						Interval:          "1m",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			bm := &storagev1alpha1.BucketMonitor{}
			err := k8sClient.Get(ctx, namespacedName, bm)
			if err == nil {
				By("cleaning up the BucketMonitor resource")
				Expect(k8sClient.Delete(ctx, bm)).To(Succeed())
			}

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
			if err == nil {
				By("cleaning up the credentials secret")
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}
		})

		It("should set phase=Error when S3 endpoint is unreachable", func() {
			By("reconciling the resource")
			reconciler := &BucketMonitorReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			// S3 is not available in the test environment — expect an error and Error phase.
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			By("verifying status is updated to Error")
			updated := &storagev1alpha1.BucketMonitor{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal("Error"))
			Expect(updated.Status.LastChecked.IsZero()).To(BeFalse())
		})
	})
})
