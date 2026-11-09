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

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gridv1alpha1 "github.com/downlinklabs/downlink/operator/api/v1alpha1"
)

// makeWorkflowWithS3Artifact builds a minimal Workflow with a playwright-report S3 artifact
// at the given bucket/key — used to verify artifactURL extraction without envtest.
func makeWorkflowWithS3Artifact(name, bucket, key string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				name: wfv1.NodeStatus{
					Outputs: &wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "playwright-report",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										S3Bucket: wfv1.S3Bucket{Bucket: bucket},
										Key:      key,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

var _ = Describe("artifactURL", func() {
	It("extracts s3:// URL from a workflow with an S3 artifact", func() {
		wf := makeWorkflowWithS3Artifact("run-abc123", "my-bucket", "httbin-synthetic/run-abc123/playwright-report")
		Expect(artifactURL(wf)).To(Equal("s3://my-bucket/httbin-synthetic/run-abc123/playwright-report"))
	})

	It("returns empty string when no playwright-report artifact exists", func() {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "run-xyz"},
			Status:     wfv1.WorkflowStatus{},
		}
		Expect(artifactURL(wf)).To(BeEmpty())
	})

	It("produces unique URLs for two runs of the same test", func() {
		wf1 := makeWorkflowWithS3Artifact(
			"httbin-synthetic-aaa111",
			"my-bucket",
			"httbin-synthetic/httbin-synthetic-aaa111/playwright-report",
		)
		wf2 := makeWorkflowWithS3Artifact(
			"httbin-synthetic-bbb222",
			"my-bucket",
			"httbin-synthetic/httbin-synthetic-bbb222/playwright-report",
		)
		url1 := artifactURL(wf1)
		url2 := artifactURL(wf2)
		Expect(url1).NotTo(BeEmpty())
		Expect(url2).NotTo(BeEmpty())
		Expect(url1).NotTo(Equal(url2))
	})
})

var _ = Describe("SyntheticTestRun Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		synthetictestrun := &gridv1alpha1.SyntheticTestRun{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind SyntheticTestRun")
			err := k8sClient.Get(ctx, typeNamespacedName, synthetictestrun)
			if err != nil && errors.IsNotFound(err) {
				resource := &gridv1alpha1.SyntheticTestRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &gridv1alpha1.SyntheticTestRun{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SyntheticTestRun")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &SyntheticTestRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
