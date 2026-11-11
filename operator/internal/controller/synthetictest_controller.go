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

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigsyaml "sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	gridv1alpha1 "github.com/downlinklabs/downlink/operator/api/v1alpha1"
)

// SyntheticTestReconciler reconciles a SyntheticTest object
type SyntheticTestReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	WorkflowBuilder WorkflowBuilder
}

// +kubebuilder:rbac:groups=downlinklabs.io,resources=synthetictests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=downlinklabs.io,resources=synthetictests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=downlinklabs.io,resources=synthetictests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=cronworkflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=workflowtaskresults,verbs=create;patch

func (r *SyntheticTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	syntheticTest := &gridv1alpha1.SyntheticTest{}
	if err := r.Get(ctx, req.NamespacedName, syntheticTest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configMapName := syntheticTest.Name + "-config"
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: syntheticTest.Namespace,
		},
		Data: map[string]string{
			"test.spec.ts": syntheticTest.Spec.Source.Inline,
		},
	}

	if err := ctrl.SetControllerReference(syntheticTest, configMap, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data = map[string]string{
			"test.spec.ts": syntheticTest.Spec.Source.Inline,
		}
		return ctrl.SetControllerReference(syntheticTest, configMap, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, err
	}

	saName := syntheticTest.Name
	if syntheticTest.Spec.ServiceAccountName != nil && *syntheticTest.Spec.ServiceAccountName != "" {
		saName = *syntheticTest.Spec.ServiceAccountName
	}
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: syntheticTest.Namespace,
		},
	}
	if err := ctrl.SetControllerReference(syntheticTest, serviceAccount, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		if syntheticTest.Spec.ArtifactStorage != nil &&
			syntheticTest.Spec.ArtifactStorage.Enabled &&
			len(syntheticTest.Spec.ArtifactStorage.ServiceAccountAnnotations) > 0 {
			if serviceAccount.Annotations == nil {
				serviceAccount.Annotations = make(map[string]string)
			}
			for k, v := range syntheticTest.Spec.ArtifactStorage.ServiceAccountAnnotations {
				serviceAccount.Annotations[k] = v
			}
		}
		return ctrl.SetControllerReference(syntheticTest, serviceAccount, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, err
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      syntheticTest.Name,
			Namespace: syntheticTest.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"workflowtaskresults"},
				Verbs:     []string{"create", "patch"},
			},
		},
	}

	if err := ctrl.SetControllerReference(syntheticTest, role, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      syntheticTest.Name,
			Namespace: syntheticTest.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     syntheticTest.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: syntheticTest.Namespace,
			},
		},
	}

	if err := ctrl.SetControllerReference(syntheticTest, rb, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	workflow, err := r.WorkflowBuilder.BuildCronWorkflow(ctx, syntheticTest)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := ctrl.SetControllerReference(syntheticTest, workflow, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	existing := &wfv1.CronWorkflow{}
	existing.Name = workflow.Name
	existing.Namespace = workflow.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, existing, func() error {
		existing.Labels = workflow.Labels
		existing.Spec = workflow.Spec
		return ctrl.SetControllerReference(syntheticTest, existing, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileArtifactRepository(ctx, syntheticTest); err != nil {
		return ctrl.Result{}, err
	}

	_ = log
	return ctrl.Result{}, nil
}

// artifactRepoS3 is the schema for the s3 section of an Argo artifact repository ConfigMap entry.
type artifactRepoS3 struct {
	Bucket          string        `json:"bucket"`
	Endpoint        string        `json:"endpoint,omitempty"`
	Region          string        `json:"region,omitempty"`
	UseSDKCreds     bool          `json:"useSDKCreds,omitempty"`
	AccessKeySecret *secretKeyRef `json:"accessKeySecret,omitempty"`
	SecretKeySecret *secretKeyRef `json:"secretKeySecret,omitempty"`
	KeyFormat       string        `json:"keyFormat"`
}

type secretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type artifactRepoConfig struct {
	S3 artifactRepoS3 `json:"s3"`
}

func (r *SyntheticTestReconciler) reconcileArtifactRepository(ctx context.Context, test *gridv1alpha1.SyntheticTest) error {
	log := logf.FromContext(ctx)
	storage := test.Spec.ArtifactStorage
	if storage == nil || !storage.Enabled {
		return nil
	}
	if storage.Bucket == "" {
		log.Info("artifactStorage.bucket is required when enabled; skipping artifact repository reconciliation", "test", test.Name)
		return nil
	}

	yamlContent, err := buildArtifactRepositoryYAML(storage)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.Name + "-artifact-repository",
			Namespace: test.Namespace,
		},
	}
	if err := ctrl.SetControllerReference(test, cm, r.Scheme); err != nil {
		return err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{
			"default": yamlContent,
		}
		return ctrl.SetControllerReference(test, cm, r.Scheme)
	})
	return err
}

func buildArtifactRepositoryYAML(storage *gridv1alpha1.ArtifactStorageSpec) (string, error) {
	const keyFormat = `{{workflow.name}}/playwright-report`

	s3 := artifactRepoS3{
		Bucket:    storage.Bucket,
		Endpoint:  storage.Endpoint,
		Region:    storage.Region,
		KeyFormat: keyFormat,
	}

	// UseSDKCreds covers EC2 instance profile, EKS IRSA (via SA annotation), and
	// any other ambient AWS credential source. ServiceAccountAnnotations also imply it.
	if storage.UseSDKCreds || len(storage.ServiceAccountAnnotations) > 0 {
		s3.UseSDKCreds = true
	} else if storage.CredentialsSecret != "" {
		s3.AccessKeySecret = &secretKeyRef{Name: storage.CredentialsSecret, Key: "accessKey"}
		s3.SecretKeySecret = &secretKeyRef{Name: storage.CredentialsSecret, Key: "secretKey"}
	}

	bytes, err := sigsyaml.Marshal(artifactRepoConfig{S3: s3})
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *SyntheticTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gridv1alpha1.SyntheticTest{}).
		Named("synthetictest").
		Complete(r)
}
