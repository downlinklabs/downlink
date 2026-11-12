package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	gridv1alpha1 "github.com/downlinklabs/downlink/operator/api/v1alpha1"
)

// fakeWorkflowBuilder implements WorkflowBuilder for use in reconciler tests.
type fakeWorkflowBuilder struct {
	buildCronWorkflowCalled int
	buildWorkflowCalled     int
	receivedTest            *gridv1alpha1.SyntheticTest
	cronWorkflowResult      *wfv1.CronWorkflow
	workflowResult          *wfv1.Workflow
	err                     error
}

func (f *fakeWorkflowBuilder) BuildCronWorkflow(_ context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.CronWorkflow, error) {
	f.buildCronWorkflowCalled++
	f.receivedTest = test
	return f.cronWorkflowResult, f.err
}

func (f *fakeWorkflowBuilder) BuildWorkflow(_ context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.Workflow, error) {
	f.buildWorkflowCalled++
	f.receivedTest = test
	return f.workflowResult, f.err
}

func buildFakeScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = wfv1.AddToScheme(s)
	_ = gridv1alpha1.AddToScheme(s)
	return s
}

var _ = Describe("SyntheticTest Controller (fake builder)", func() {
	const (
		testName      = "test-synthetic"
		testNamespace = "default"
	)

	var (
		testScheme *runtime.Scheme
		sa         string
	)

	BeforeEach(func() {
		testScheme = buildFakeScheme()
		sa = "test-sa"
	})

	newSyntheticTest := func() *gridv1alpha1.SyntheticTest {
		return &gridv1alpha1.SyntheticTest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				UID:       types.UID("test-uid-1234"),
			},
			Spec: gridv1alpha1.SyntheticTestSpec{
				Schedule:           "*/5 * * * *",
				ServiceAccountName: &sa,
				Entrypoint:         "run",
				Source: gridv1alpha1.SyntheticTestSource{
					Inline: "import { test } from '@playwright/test'",
				},
			},
		}
	}

	newCronWorkflow := func() *wfv1.CronWorkflow {
		return &wfv1.CronWorkflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: wfv1.CronWorkflowSpec{
				Schedules: []string{"*/5 * * * *"},
			},
		}
	}

	buildReconciler := func(builder WorkflowBuilder, syntheticTest *gridv1alpha1.SyntheticTest) *SyntheticTestReconciler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(syntheticTest).
			Build()
		return &SyntheticTestReconciler{
			Client:          fakeClient,
			Scheme:          testScheme,
			WorkflowBuilder: builder,
		}
	}

	reconcileRequest := func() reconcile.Request {
		return reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      testName,
				Namespace: testNamespace,
			},
		}
	}

	Context("create path", func() {
		It("calls BuildCronWorkflow with the SyntheticTest and applies the returned CronWorkflow", func() {
			st := newSyntheticTest()
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			Expect(fake.buildCronWorkflowCalled).To(Equal(1))
			Expect(fake.receivedTest.Name).To(Equal(testName))

			applied := &wfv1.CronWorkflow{}
			err = r.Client.Get(context.Background(), types.NamespacedName{Name: testName, Namespace: testNamespace}, applied)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("error path", func() {
		It("returns the builder error and does not apply any CronWorkflow", func() {
			st := newSyntheticTest()
			builderErr := fmt.Errorf("construction failed")
			fake := &fakeWorkflowBuilder{err: builderErr}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).To(MatchError(builderErr))

			missing := &wfv1.CronWorkflow{}
			getErr := r.Client.Get(context.Background(), types.NamespacedName{Name: testName, Namespace: testNamespace}, missing)
			Expect(getErr).To(HaveOccurred())
		})
	})

	Context("update path", func() {
		It("calls BuildCronWorkflow on each reconcile", func() {
			st := newSyntheticTest()
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			_, err = r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			Expect(fake.buildCronWorkflowCalled).To(Equal(2))
		})
	})

	Context("artifact storage — static credentials", func() {
		It("creates an artifact-repository ConfigMap with accessKeySecret and secretKeySecret", func() {
			st := newSyntheticTest()
			st.Spec.ArtifactStorage = &gridv1alpha1.ArtifactStorageSpec{
				Enabled:           true,
				Bucket:            "test-bucket",
				Region:            "us-east-2",
				CredentialsSecret: "my-s3-creds",
			}
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Name:      testName + "-artifact-repository",
				Namespace: testNamespace,
			}, cm)
			Expect(err).NotTo(HaveOccurred())

			data := cm.Data["default"]
			Expect(data).To(ContainSubstring("bucket: test-bucket"))
			Expect(data).To(ContainSubstring("region: us-east-2"))
			Expect(data).To(ContainSubstring("name: my-s3-creds"))
			Expect(data).To(ContainSubstring("key: accessKey"))
			Expect(data).To(ContainSubstring("key: secretKey"))
			Expect(data).NotTo(ContainSubstring("useSDKCreds"))
			Expect(data).To(ContainSubstring("{{workflow.name}}/playwright-report"))
		})

		It("skips ConfigMap creation when bucket is empty", func() {
			st := newSyntheticTest()
			st.Spec.ArtifactStorage = &gridv1alpha1.ArtifactStorageSpec{
				Enabled: true,
				Bucket:  "",
			}
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Name:      testName + "-artifact-repository",
				Namespace: testNamespace,
			}, cm)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("artifact storage — IAM delegation", func() {
		It("creates an artifact-repository ConfigMap with useSDKCreds and no credential keys", func() {
			st := newSyntheticTest()
			st.Spec.ArtifactStorage = &gridv1alpha1.ArtifactStorageSpec{
				Enabled: true,
				Bucket:  "iam-bucket",
				ServiceAccountAnnotations: map[string]string{
					"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789:role/downlink-uploader",
				},
			}
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Name:      testName + "-artifact-repository",
				Namespace: testNamespace,
			}, cm)
			Expect(err).NotTo(HaveOccurred())

			data := cm.Data["default"]
			Expect(data).To(ContainSubstring("useSDKCreds: true"))
			Expect(data).NotTo(ContainSubstring("accessKeySecret"))
			Expect(data).NotTo(ContainSubstring("secretKeySecret"))
		})

		It("merges ServiceAccountAnnotations onto the created ServiceAccount", func() {
			st := newSyntheticTest()
			roleArn := "arn:aws:iam::123456789:role/downlink-uploader"
			st.Spec.ArtifactStorage = &gridv1alpha1.ArtifactStorageSpec{
				Enabled: true,
				Bucket:  "iam-bucket",
				ServiceAccountAnnotations: map[string]string{
					"eks.amazonaws.com/role-arn": roleArn,
				},
			}
			cw := newCronWorkflow()
			fake := &fakeWorkflowBuilder{cronWorkflowResult: cw}
			r := buildReconciler(fake, st)

			_, err := r.Reconcile(context.Background(), reconcileRequest())
			Expect(err).NotTo(HaveOccurred())

			svcAcct := &corev1.ServiceAccount{}
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Name:      sa,
				Namespace: testNamespace,
			}, svcAcct)
			Expect(err).NotTo(HaveOccurred())
			Expect(svcAcct.Annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleArn))
		})
	})
})
