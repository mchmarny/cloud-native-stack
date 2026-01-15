package agent

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

// ensureJob deletes any existing Job and creates a fresh one.
func (d *Deployer) ensureJob(ctx context.Context) error {
	// Delete existing Job if present
	propagationPolicy := metav1.DeletePropagationForeground
	err := d.clientset.BatchV1().Jobs(d.config.Namespace).Delete(
		ctx,
		d.config.JobName,
		metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete existing Job: %w", err)
	}

	// Wait for Job to be fully deleted
	jobExisted := err == nil // Job existed and was deleted
	if jobExisted {
		if waitErr := d.waitForJobDeletion(ctx); waitErr != nil {
			return fmt.Errorf("timeout waiting for Job deletion: %w", waitErr)
		}
	}

	// Create fresh Job
	job := d.buildJob()
	_, err = d.clientset.BatchV1().Jobs(d.config.Namespace).
		Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	return nil
}

// buildJob constructs the Job specification.
func (d *Deployer) buildJob() *batchv1.Job {
	// Build command arguments (directly invoke binary without shell)
	args := []string{"snapshot", "-o", d.config.Output}
	if d.config.Debug {
		args = []string{"--debug", "--log-json", "snapshot", "-o", d.config.Output}
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.config.JobName,
			Namespace: d.config.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "cns",
			},
		},
		Spec: batchv1.JobSpec{
			Completions:             ptr.To(int32(1)),
			Parallelism:             ptr.To(int32(1)),
			CompletionMode:          ptr.To(batchv1.NonIndexedCompletion),
			BackoffLimit:            ptr.To(int32(0)),
			TTLSecondsAfterFinished: ptr.To(int32(3600)),
			ActiveDeadlineSeconds:   ptr.To(int64(18000)), // 5 hours
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "cns",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.config.ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					HostPID:            true,
					HostNetwork:        true,
					HostIPC:            true,
					NodeSelector:       d.config.NodeSelector,
					Tolerations:        d.config.Tolerations,
					ImagePullSecrets:   toLocalObjectReferences(d.config.ImagePullSecrets),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:           ptr.To(int64(0)),
						RunAsGroup:          ptr.To(int64(0)),
						FSGroup:             ptr.To(int64(0)),
						FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
					},
					Containers: []corev1.Container{
						{
							Name:    "cns",
							Image:   d.config.Image,
							Command: []string{"/ko-app/cnsctl"},
							Args:    args,
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:              mustParseQuantity("1"),
									corev1.ResourceMemory:           mustParseQuantity("4Gi"),
									corev1.ResourceEphemeralStorage: mustParseQuantity("2Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:              mustParseQuantity("2"),
									corev1.ResourceMemory:           mustParseQuantity("8Gi"),
									corev1.ResourceEphemeralStorage: mustParseQuantity("4Gi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged:               ptr.To(true),
								RunAsUser:                ptr.To(int64(0)),
								RunAsGroup:               ptr.To(int64(0)),
								AllowPrivilegeEscalation: ptr.To(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_ADMIN", "SYS_CHROOT"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "run-systemd",
									MountPath: "/run/systemd",
									ReadOnly:  true,
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "run-systemd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/systemd",
									Type: ptr.To(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

// deleteJob deletes the Job.
func (d *Deployer) deleteJob(ctx context.Context) error {
	propagationPolicy := metav1.DeletePropagationForeground
	err := d.clientset.BatchV1().Jobs(d.config.Namespace).Delete(
		ctx,
		d.config.JobName,
		metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	)
	return ignoreNotFound(err)
}

// waitForJobDeletion waits for the Job to be fully deleted.
func (d *Deployer) waitForJobDeletion(ctx context.Context) error {
	timeout := 30 * time.Second
	return wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, timeout, true,
		func(ctx context.Context) (bool, error) {
			_, err := d.clientset.BatchV1().Jobs(d.config.Namespace).
				Get(ctx, d.config.JobName, metav1.GetOptions{})
			if ignoreNotFound(err) == nil {
				return true, nil // Job deleted successfully
			}
			if err != nil {
				return false, err
			}
			return false, nil // Job still exists, keep waiting
		},
	)
}

// mustParseQuantity parses a resource quantity or panics.
func mustParseQuantity(s string) resource.Quantity {
	q := resource.MustParse(s)
	return q
}

// toLocalObjectReferences converts a slice of secret names to LocalObjectReferences.
func toLocalObjectReferences(names []string) []corev1.LocalObjectReference {
	if len(names) == 0 {
		return nil
	}
	refs := make([]corev1.LocalObjectReference, len(names))
	for i, name := range names {
		refs[i] = corev1.LocalObjectReference{Name: name}
	}
	return refs
}
