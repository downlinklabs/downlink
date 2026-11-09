package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	gridv1alpha1 "github.com/downlinklabs/downlink/operator/api/v1alpha1"
)

const (
	defaultName    = "playwright"
	testsMountPath = "/app/tests"
)

var defaultCommand = []string{"npx", "playwright", "test"}

func serviceAccountName(test *gridv1alpha1.SyntheticTest) string {
	if test.Spec.ServiceAccountName != nil && *test.Spec.ServiceAccountName != "" {
		return *test.Spec.ServiceAccountName
	}
	return test.Name
}

// WorkflowBuilder inflates a SyntheticTest into Argo workflow objects.
// Implementations must be safe to call concurrently on the same instance.
type WorkflowBuilder interface {
	BuildCronWorkflow(ctx context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.CronWorkflow, error)
	BuildWorkflow(ctx context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.Workflow, error)
}

// DefaultWorkflowBuilder is the production WorkflowBuilder.
type DefaultWorkflowBuilder struct {
	RunnerImage    string
	PushgatewayURL string
}

func (b *DefaultWorkflowBuilder) BuildCronWorkflow(_ context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.CronWorkflow, error) {
	if test.Spec.Source.Inline == "" {
		return nil, fmt.Errorf("SyntheticTest %s/%s: spec.source.inline is required", test.Namespace, test.Name)
	}
	if test.Spec.Schedule == "" {
		return nil, fmt.Errorf("SyntheticTest %s/%s: spec.schedule is required for a CronWorkflow", test.Namespace, test.Name)
	}

	workflowSpec := b.buildWorkflowSpec(test)

	return &wfv1.CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.Name,
			Namespace: test.Namespace,
			Labels: map[string]string{
				"the-grid.io/synthetic-test": test.Name,
			},
		},
		Spec: wfv1.CronWorkflowSpec{
			Schedules: []string{test.Spec.Schedule},
			WorkflowMetadata: &metav1.ObjectMeta{
				Labels: map[string]string{
					"the-grid.io/synthetic-test": test.Name,
				},
			},
			WorkflowSpec: workflowSpec,
		},
	}, nil
}

func (b *DefaultWorkflowBuilder) BuildWorkflow(_ context.Context, test *gridv1alpha1.SyntheticTest) (*wfv1.Workflow, error) {
	if test.Spec.Source.Inline == "" {
		return nil, fmt.Errorf("SyntheticTest %s/%s: spec.source.inline is required", test.Namespace, test.Name)
	}

	workflowSpec := b.buildWorkflowSpec(test)

	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.Name,
			Namespace: test.Namespace,
			Labels: map[string]string{
				"the-grid.io/synthetic-test": test.Name,
			},
		},
		Spec: workflowSpec,
	}, nil
}

// buildWorkflowSpec constructs the shared WorkflowSpec from a SyntheticTest.
func (b *DefaultWorkflowBuilder) buildWorkflowSpec(test *gridv1alpha1.SyntheticTest) wfv1.WorkflowSpec {
	configMapName := test.Name + "-config"

	container := test.Spec.Container
	if container.Image == "" && b.RunnerImage != "" {
		container.Image = b.RunnerImage
	}
	if b.PushgatewayURL != "" {
		hasPushgatewayURL := false
		for _, env := range container.Env {
			if env.Name == "PUSHGATEWAY_URL" {
				hasPushgatewayURL = true
				break
			}
		}
		if !hasPushgatewayURL {
			container.Env = append([]corev1.EnvVar{{Name: "PUSHGATEWAY_URL", Value: b.PushgatewayURL}}, container.Env...)
		}
	}
	if len(container.Command) == 0 {
		container.Command = defaultCommand
	}
	if container.Name == "" {
		container.Name = defaultName
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      test.Name + "-source",
		MountPath: testsMountPath,
	})

	template := wfv1.Template{
		Name:      "run",
		Container: &container,
	}

	artifactStorageEnabled := test.Spec.ArtifactStorage != nil && test.Spec.ArtifactStorage.Enabled
	if artifactStorageEnabled {
		template.Outputs = wfv1.Outputs{
			Artifacts: wfv1.Artifacts{
				{
					Name: "playwright-report",
					Path: "/app/playwright-report",
				},
			},
		}
	}

	workflowSpec := wfv1.WorkflowSpec{
		ServiceAccountName: serviceAccountName(test),
		Entrypoint:         test.Spec.Entrypoint,
		Templates:          []wfv1.Template{template},
		Volumes: []corev1.Volume{
			{
				Name: test.Name + "-source",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
					},
				},
			},
		},
	}

	if artifactStorageEnabled {
		workflowSpec.ArtifactRepositoryRef = &wfv1.ArtifactRepositoryRef{
			ConfigMap: test.Name + "-artifact-repository",
			Key:       "default",
		}
	}

	return workflowSpec
}
