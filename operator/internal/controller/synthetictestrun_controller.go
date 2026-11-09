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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	gridv1alpha1 "github.com/downlinklabs/downlink/operator/api/v1alpha1"
)

// +kubebuilder:rbac:groups=thegrid.io,resources=synthetictests,verbs=get;list;watch
// +kubebuilder:rbac:groups=thegrid.io,resources=synthetictestruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=thegrid.io,resources=synthetictestruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=thegrid.io,resources=synthetictestruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch

type SyntheticTestRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SyntheticTestRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	workflow := &wfv1.Workflow{}
	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !workflow.Status.Phase.Completed() {
		return ctrl.Result{}, nil
	}

	testName := workflow.Labels["downlinklabs.io/synthetic-test"]

	syntheticTest := &gridv1alpha1.SyntheticTest{}
	if err := r.Get(ctx, types.NamespacedName{Name: testName, Namespace: workflow.Namespace}, syntheticTest); err != nil {
		// If the SyntheticTest is gone, nothing to own — skip silently
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	run := &gridv1alpha1.SyntheticTestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
		Spec: gridv1alpha1.SyntheticTestRunSpec{
			Name:         testName,
			WorkflowName: workflow.Name,
			StartedAt:    &workflow.Status.StartedAt,
		},
	}

	if err := ctrl.SetControllerReference(syntheticTest, run, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, run); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
		// Object already exists — Get it to obtain the current ResourceVersion for the status update.
		if err := r.Get(ctx, types.NamespacedName{Name: run.Name, Namespace: run.Namespace}, run); err != nil {
			return ctrl.Result{}, err
		}
	}
	// run is now either the just-created object (populated by the API server response) or the fetched existing one.

	run.Status.Phase = string(workflow.Status.Phase)
	run.Status.Message = workflow.Status.Message
	if !workflow.Status.FinishedAt.IsZero() {
		t := workflow.Status.FinishedAt.DeepCopy()
		run.Status.FinishedAt = t
	}
	if url := artifactURL(workflow); url != "" {
		run.Status.ArtifactURL = url
	}

	if err := r.Status().Update(ctx, run); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled SyntheticTestRun", "workflow", workflow.Name, "phase", workflow.Status.Phase)
	return ctrl.Result{}, nil
}

func artifactURL(workflow *wfv1.Workflow) string {
	for _, node := range workflow.Status.Nodes {
		if node.Outputs == nil {
			continue
		}
		for _, a := range node.Outputs.Artifacts {
			if a.Name != "playwright-report" {
				continue
			}
			if a.S3 != nil {
				return fmt.Sprintf("s3://%s/%s", a.S3.Bucket, a.S3.Key)
			}
			if a.GCS != nil {
				return fmt.Sprintf("gs://%s/%s", a.GCS.Bucket, a.GCS.Key)
			}
			if a.Azure != nil {
				return fmt.Sprintf("%s/%s/%s",
					a.Azure.Endpoint, a.Azure.Container, a.Azure.Blob)
			}
			if a.OSS != nil {
				return fmt.Sprintf("oss://%s/%s", a.OSS.Bucket, a.OSS.Key)
			}
		}
	}
	return ""
}

func (r *SyntheticTestRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wfv1.Workflow{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(
				func(object client.Object) bool {
					labels := object.GetLabels()
					_, ok := labels["downlinklabs.io/synthetic-test"]
					return ok
				},
			),
		)).
		Named("synthetictestrun").
		Complete(r)
}
