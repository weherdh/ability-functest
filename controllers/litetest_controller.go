/*
Copyright 2022.

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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	testv1 "github.com/weherdh/ability-functest/api/v1"
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// LiteTestReconciler reconciles a LiteTest object
type LiteTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=test.ability.app,resources=litetests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.ability.app,resources=litetests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.ability.app,resources=litetests/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LiteTest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LiteTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	const MOK = "meta.ownerReferences.kind"
	const NodeSelector = "kubernetes.io/hostname"
	const RO = "ReportOnline"
	const TF = "TestFailed"
	const TR = "TestRunning"
	const TD = "TestFinished"
	liteTest := &testv1.LiteTest{}
	if err := r.Get(ctx, req.NamespacedName, liteTest); err != nil {
		log.Error(err, "unable to fetch LiteTest")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	testEnv := liteTest.Spec.LiteEnv
	constructJobForLiteTest := func(liteTest *testv1.LiteTest) (*batchv1.Job, error) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        liteTest.Name,
				Namespace:   liteTest.Namespace,
			},
			Spec: *liteTest.Spec.JobTemplate.Spec.DeepCopy(),
		}
		for k, v := range liteTest.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		for k, v := range liteTest.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		job.Spec.Template.Spec.Containers[0].Env = testEnv
		if err := ctrl.SetControllerReference(liteTest, job, r.Scheme); err != nil {
			return nil, err
		}
		return job, nil
	}
	testJob := &batchv1.Job{}
	if err := r.Get(ctx, req.NamespacedName, testJob); err != nil {
		if errors.IsNotFound(err) {
			job, err := constructJobForLiteTest(liteTest)
			if err != nil {
				log.Error(err, "Unable to construct test job for Lite test")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Unable to create test job for Lite test")
				return ctrl.Result{}, err
			}
		}
		log.Error(err, "unable to fetch Lite testing job")
		return ctrl.Result{}, err
	}
	reportImage := liteTest.Spec.ReportImage
	generateReportForLiteTest := func(liteTest *testv1.LiteTest) (*appv1.Deployment, error) {
		dep := &appv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Labels:    make(map[string]string),
				Name:      liteTest.Name,
				Namespace: liteTest.Namespace,
			},
			Spec: appv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: reportImage,
							Name:  liteTest.Name,
							Ports: []corev1.ContainerPort{{
								ContainerPort: 80,
								Name:          liteTest.Name,
							}},
						}},
					},
				},
			},
		}
		for k, v := range liteTest.Spec.JobTemplate.Labels {
			dep.Labels[k] = v
			dep.Spec.Template.Labels[k] = v
		}
		if err := controllerutil.SetControllerReference(liteTest, dep, r.Scheme); err != nil {
			return nil, err
		}
		return dep, nil
	}
	reportDeploy := &appv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, reportDeploy); err != nil {
		if errors.IsNotFound(err) {
			dep, err := generateReportForLiteTest(liteTest)
			if err != nil {
				log.Error(err, "Unable to construct test report for Lite test")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, dep); err != nil {
				log.Error(err, "Unable to generate test report for Lite test")
				return ctrl.Result{}, err
			}
		}
		log.Error(err, "unable to fetch Lite Testing report")
		return ctrl.Result{}, err
	}
	jobPods := &corev1.PodList{}
	if err := r.List(ctx, jobPods, client.InNamespace(req.Namespace), client.MatchingFields{MOK: testJob.OwnerReferences[0].Kind}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}
	deployPods := &corev1.PodList{}
	if err := r.List(ctx, deployPods, client.InNamespace(req.Namespace), client.MatchingFields{MOK: reportDeploy.OwnerReferences[0].Kind}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	isJobFinished := func(job *batchv1.Job) (bool, batchv1.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}
		return false, ""
	}
	//The following logic what I designed is
	//1. Judge whether the job finished, if error, return err
	//2. If done, get the node from the pod, patch the deployment to deploy report with the same node
	//3. Compare the deployment->pods node with job->pod node, if yes, do not patch
	if jobStatus, jobType := isJobFinished(testJob); jobStatus {
		if jobType == batchv1.JobComplete {
			jobNode := jobPods.Items[0].Spec.NodeName
			deployNode := deployPods.Items[0].Spec.NodeName
			liteTest.Status.TestStatus = TD
			if jobNode != deployNode {
				reportDeploy.Spec.Template.Spec.NodeSelector = map[string]string{NodeSelector: jobNode}
				if err := r.Update(ctx, reportDeploy); err != nil {
					return ctrl.Result{}, err
				}
				//patch := client.MergeFrom(reportDeploy.DeepCopy())
				//r.Patch(ctx, reportDeploy, patch)
				liteTest.Status.TestStatus = RO
			}
		} else if jobType == batchv1.JobFailed {
			replica := int32(0)
			if reportDeploy.Spec.Replicas != &replica {
				if err := r.Update(ctx, reportDeploy); err != nil {
					log.Error(err, "Job failed and Unable to disable report")
					return ctrl.Result{}, err
				}
			}
			liteTest.Status.TestStatus = TF
		} else {
			liteTest.Status.TestStatus = TR
		}
	}
	if err := r.Status().Update(ctx, liteTest); err != nil {
		log.Error(err, "unable to update LiteTest status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LiteTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1.LiteTest{}).
		Owns(&batchv1.Job{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
