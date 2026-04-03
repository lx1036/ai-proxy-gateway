package jobset

import (
	"context"
	"errors"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"maps"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"slices"
	"strconv"
	"sync"

	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

/**
JobSet Controller 作为一个非常好的 demo 示例架构：
JobSet -> batchv1.Job -> Pods


*/

type JobSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewJobSetReconciler(mgr manager.Manager) *JobSetReconciler {
	return &JobSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("jobset-controller"),
	}
}

func (r *JobSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jobset.JobSet{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *JobSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	jobSet := &jobset.JobSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, jobSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !jobSet.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// INFO: 检查只 Reconcile 当前控制器管理的 JobSet.
	//  .Spec.ManagedBy 字段，很常用的方法
	if controllerName := jobSet.Spec.ManagedBy; controllerName != nil && *controllerName != jobset.JobSetControllerName {
		klog.Infof("JobSet %s/%s is not managed by %s, skip reconcile.", jobSet.Namespace, jobSet.Name, *controllerName)
		return ctrl.Result{}, nil
	}

	// INFO: 1. get jobset 所属的所有 batchv1.jobs

	// INFO: debug 这里的写法，使用了 client.MatchingFields{".metadata.controller": string(jobSet.UID)}
	var jobList batchv1.JobList
	if err := r.List(ctx, &jobList, client.InNamespace(jobSet.Namespace), client.MatchingFields{".metadata.controller": string(jobSet.UID)}); err != nil {
		klog.Errorf("JobSet %s/%s failed to list jobs: %v", jobSet.Namespace, jobSet.Name, err)
		return ctrl.Result{}, err
	}
	var activeJobs []*batchv1.Job
	var failedJobs []*batchv1.Job
	var succeededJobs []*batchv1.Job
	var previousJobs []*batchv1.Job
	for _, item := range jobList.Items {
		restarts, err := strconv.Atoi(item.Labels["jobset.sigs.k8s.io/restart-attempt"])
		if err != nil {
			klog.Errorf("JobSet %s/%s failed to parse job %s/%s restart count: %v", jobSet.Namespace, jobSet.Name, item.Namespace, item.Name, err)
			previousJobs = append(previousJobs, &item)
			continue
		}
		if int32(restarts) < jobSet.Status.Restarts { // jobSet.Status.Restarts is target max restarts. TODO: 有点奇怪，不应该是 restarts 是 target 么？
			klog.Infof("JobSet %s/%s found previous job %s/%s for restart %d less then target max restarts %d", jobSet.Namespace, jobSet.Name, item.Namespace, item.Name, jobSet.Status.Restarts, restarts)
			previousJobs = append(previousJobs, &item)
			continue
		}

		for _, condition := range item.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
				succeededJobs = append(succeededJobs, &item)
			} else if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				failedJobs = append(failedJobs, &item)
			} else {
				activeJobs = append(activeJobs, &item)
			}
		}
	}

	// INFO: 2. jobset 已经删除，则清理所有 active jobs 
	if IsJobSetFinished(jobSet) {
		if err := r.deleteJobs(ctx, activeJobs); err != nil {
			klog.Errorf("JobSet %s/%s failed to delete jobs: %v", jobSet.Namespace, jobSet.Name, err)
			return ctrl.Result{}, err
		}

		// TODO: ttl after finished policy

		return ctrl.Result{}, nil
	}

	// INFO: 3. 删除重试次数超过 target restarts 的 jobs
	if err := r.deleteJobs(ctx, previousJobs); err != nil {
		klog.Errorf("JobSet %s/%s failed to delete jobs: %v", jobSet.Namespace, jobSet.Name, err)
		return ctrl.Result{}, err
	}

	// execute failed jobs policy
	if len(failedJobs) > 0 {

	}

	// execute succeeded jobs policy
	if len(succeededJobs) > 0 {

	}

	// INFO: 4. 创建 batchv1.jobs
	for _, replicatedJob := range jobSet.Spec.ReplicatedJobs {
		jobs := constructJobsFromReplicatedJob(jobSet, replicatedJob)

		r.createJobs(ctx, jobs)

	}

	klog.Infof("JobSet Reconcile completed.")
	return ctrl.Result{}, nil
}

func (r *JobSetReconciler) deleteJobs(ctx context.Context, jobs []*batchv1.Job) error {
	var errs []error
	lock := &sync.RWMutex{}
	workqueue.ParallelizeUntil(ctx, 50, len(jobs), func(i int) {
		deletedJob := jobs[i]
		if deletedJob.DeletionTimestamp != nil {
			return
		}

		if err := r.Delete(ctx, deletedJob, client.PropagationPolicy(metav1.DeletePropagationForeground)); client.IgnoreNotFound(err) != nil {
			lock.Lock()
			defer lock.Unlock()
			err = fmt.Errorf("failed to delete job %s/%s: %v", deletedJob.Namespace, deletedJob.Name, err)
			errs = append(errs, err)
		}
	})

	return errors.Join(errs...)

	/* 最简单粗暴的删除 jobs 的方法
	for _, job := range jobs {
		if err := r.Delete(ctx, job); err != nil {
			return err
		}
	}*/
}

func IsJobSetFinished(jobSet *jobset.JobSet) bool {
	for _, condition := range jobSet.Status.Conditions {
		if (condition.Type == string(jobset.JobSetCompleted) || condition.Type == string(jobset.JobSetFailed)) && condition.Status == metav1.ConditionTrue {
			return true
		}
	}

	return false
}

func constructJobsFromReplicatedJob(jobSet *jobset.JobSet, replicatedJob *jobset.ReplicatedJob) []*batchv1.Job {
	var jobs []*batchv1.Job
	for jobIndex := 0; jobIndex < int(replicatedJob.Replicas); jobIndex++ {
		jobName := constructJobName(jobSet, replicatedJob, jobIndex)
		if !shouldCreateJob(jobName) {
			continue
		}

		job := constructJob(jobSet, replicatedJob, jobIndex)
		jobs = append(jobs, job)
	}

	return jobs
}

func constructJobName(jobSet *jobset.JobSet, replicatedJob *jobset.ReplicatedJob, jobIndex int) string {
	return fmt.Sprintf("%s-%s-%d", jobSet.Name, replicatedJob.Name, jobIndex)
}

func constructJob(jobSet *jobset.JobSet, replicatedJob *jobset.ReplicatedJob, jobIndex int) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        constructJobName(jobSet, replicatedJob, jobIndex),
			Namespace:   jobSet.Namespace,
			Labels:      maps.Clone(replicatedJob.Template.Labels),
			Annotations: maps.Clone(replicatedJob.Template.Annotations),
		},
		Spec:   *replicatedJob.Template.Spec.DeepCopy(),
		Status: batchv1.JobStatus{},
	}

	return job
}

func shouldCreateJob(activeJobs, failedJobs, succeededJobs, previousJobs []*batchv1.Job, jobName string) bool {
	for _, job := range slices.Concat(activeJobs, failedJobs, succeededJobs, previousJobs) {
		if job.Name == jobName {
			return false
		}
	}

	return true
}
