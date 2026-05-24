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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	storagev1alpha1 "github.com/a1010s/s3-monitor-operator/api/v1alpha1"
	"github.com/a1010s/s3-monitor-operator/internal/metrics"
	s3client "github.com/a1010s/s3-monitor-operator/internal/s3"
)

// BucketMonitorReconciler reconciles a BucketMonitor object
type BucketMonitorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=storage.a1010s.io,resources=bucketmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.a1010s.io,resources=bucketmonitors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.a1010s.io,resources=bucketmonitors/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *BucketMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch BucketMonitor resource
	var bm storagev1alpha1.BucketMonitor
	if err := r.Get(ctx, req.NamespacedName, &bm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get bucketmonitor: %w", err)
	}

	interval := 5 * time.Minute
	if bm.Spec.Interval != "" {
		if d, err := time.ParseDuration(bm.Spec.Interval); err == nil {
			interval = d
		}
	}

	// setStatus is a helper to write status back; called on every path
	setStatus := func(phase, message string) {
		bm.Status.Phase = phase
		bm.Status.Message = message
		bm.Status.LastChecked = metav1.Now()
		if err := r.Status().Update(ctx, &bm); err != nil {
			log.Error(err, "failed to update status")
		}
	}

	// 2. Read credentials from secret
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      bm.Spec.CredentialsSecret,
		Namespace: req.Namespace,
	}, &secret); err != nil {
		setStatus("Error", fmt.Sprintf("credentials secret: %v", err))
		return ctrl.Result{RequeueAfter: interval}, fmt.Errorf("get secret: %w", err)
	}

	accessKey := string(secret.Data["AWS_ACCESS_KEY_ID"])
	secretKey := string(secret.Data["AWS_SECRET_ACCESS_KEY"])

	// 3+4. Build S3 client and list objects
	stats, err := s3client.GetBucketStats(ctx, bm.Spec.Endpoint, bm.Spec.Bucket, accessKey, secretKey)
	if err != nil {
		metrics.LastScrapeSuccess.WithLabelValues(bm.Spec.Bucket, req.Namespace).Set(0)
		setStatus("Error", fmt.Sprintf("s3 scrape: %v", err))
		return ctrl.Result{RequeueAfter: interval}, fmt.Errorf("get bucket stats: %w", err)
	}

	// 5. Update prometheus metrics
	metrics.BucketSizeBytes.WithLabelValues(bm.Spec.Bucket, req.Namespace).Set(float64(stats.TotalSizeBytes))
	metrics.BucketObjectCount.WithLabelValues(bm.Spec.Bucket, req.Namespace).Set(float64(stats.ObjectCount))
	metrics.LastScrapeSuccess.WithLabelValues(bm.Spec.Bucket, req.Namespace).Set(1)

	// 6. Fire event if threshold exceeded
	if bm.Spec.AlertThresholdMB != nil {
		thresholdBytes := *bm.Spec.AlertThresholdMB * 1024 * 1024
		if stats.TotalSizeBytes > thresholdBytes {
			r.Recorder.Eventf(&bm, corev1.EventTypeWarning, "ThresholdExceeded",
				"bucket %s size %s exceeds threshold of %dMB",
				bm.Spec.Bucket, humanSize(stats.TotalSizeBytes), *bm.Spec.AlertThresholdMB)
		}
	}

	// 7. Update status
	bm.Status.TotalSizeBytes = stats.TotalSizeBytes
	bm.Status.ObjectCount = stats.ObjectCount
	bm.Status.TotalSizeHuman = humanSize(stats.TotalSizeBytes)
	setStatus("Ready", "")

	// 8. Requeue after interval
	return ctrl.Result{RequeueAfter: interval}, nil
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.BucketMonitor{}).
		Named("bucketmonitor").
		Complete(r)
}
