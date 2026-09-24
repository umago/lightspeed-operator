/*
Copyright 2025.

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
	"sync/atomic"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
)

var _ = ginkgo.Describe("OpenStackLightspeed Controller", func() {
	ginkgo.Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		openstacklightspeed := &apiv1beta1.OpenStackLightspeed{}

		ginkgo.BeforeEach(func() {
			ginkgo.By("creating the custom resource for the Kind OpenStackLightspeed")
			err := k8sClient.Get(ctx, typeNamespacedName, openstacklightspeed)
			if err != nil && errors.IsNotFound(err) {
				resource := &apiv1beta1.OpenStackLightspeed{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: apiv1beta1.OpenStackLightspeedSpec{
						OpenStackLightspeedCore: apiv1beta1.OpenStackLightspeedCore{
							Lightspeed: apiv1beta1.OpenStackLightspeedConfigSpec{DefaultModel: "test-model"},
							Models: []apiv1beta1.OpenStackLightspeedModelSpec{{
								Name:            "test-model",
								LLMEndpoint:     "https://example.com/llm",
								LLMEndpointType: OpenAIProviderName,
								LLMCredentials:  "test-llm-credentials",
								ModelName:       "test-model",
							}},
						},
					},
				}
				gomega.Expect(k8sClient.Create(ctx, resource)).To(gomega.Succeed())
			}
		})

		ginkgo.AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &apiv1beta1.OpenStackLightspeed{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Cleanup the specific resource instance OpenStackLightspeed")
			gomega.Expect(k8sClient.Delete(ctx, resource)).To(gomega.Succeed())
		})
		ginkgo.It("should successfully reconcile the resource", func() {
			ginkgo.By("Reconciling the created resource")
			controllerReconciler := &OpenStackLightspeedReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				DynamicWatchCRD: DynamicWatchCRD{
					OpenStackControlPlaneGVK():         new(atomic.Bool),
					KeystoneApplicationCredentialGVK(): new(atomic.Bool),
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
