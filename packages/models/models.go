package models

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// JobSpec wraps the user-submitted Job with metadata.
type JobSpec struct {
	Name     string                 `json:"name"`
	Priority int                    `json:"priority"`
	Job      batchv1.Job            `json:"job"`
	Template corev1.PodTemplateSpec `json:"template"`
}

type JobStatus string

const (
	Pending   JobStatus = "pending"
	Running   JobStatus = "running"
	Completed JobStatus = "completed"
	Failed    JobStatus = "failed"
)

type JobInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Priority   int        `json:"priority"`
	Status     JobStatus  `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
}
