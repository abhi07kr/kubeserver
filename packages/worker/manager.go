package worker

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/abhi07kr/kubeserver/packages/models"
	"github.com/abhi07kr/kubeserver/packages/queue"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Manager struct {
	workers   int
	clientset *kubernetes.Clientset
	pq        *queue.PriorityQueue
	logger    *slog.Logger
	namespace string

	running map[string]models.JobInfo
	lock    sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewManager(workers int, clientset *kubernetes.Clientset, pq *queue.PriorityQueue, logger *slog.Logger, namespace string) *Manager {
	m := &Manager{
		workers:   workers,
		clientset: clientset,
		pq:        pq,
		logger:    logger,
		namespace: namespace,
		running:   make(map[string]models.JobInfo),
		stopCh:    make(chan struct{}),
	}
	m.Start()
	return m
}

func (m *Manager) Start() {
	m.logger.Info("Starting manager", slog.Int("workers", m.workers), slog.String("namespace", m.namespace))

	m.wg.Add(1)
	go m.startJobInformer()

	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.workerLoop(i)
	}
}

func (m *Manager) Stop() {
	m.logger.Info("Stopping manager")
	close(m.stopCh)
	m.pq.Close()
	m.wg.Wait()
	m.logger.Info("All workers and informer stopped")
}

func (m *Manager) workerLoop(id int) {
	defer m.wg.Done()
	m.logger.Info("Worker started", slog.Int("id", id))
	for {
		select {
		case <-m.stopCh:
			m.logger.Info("Worker stopping", slog.Int("id", id))
			return
		default:
		}

		jobItem := m.pq.DequeueBlocking()
		if jobItem == nil {
			return
		}

		m.logger.Info("Dequeued job", slog.String("jobID", jobItem.ID), slog.String("name", jobItem.Spec.Name))

		now := time.Now()
		m.lock.Lock()
		m.running[jobItem.ID] = models.JobInfo{
			ID:        jobItem.ID,
			Name:      jobItem.Spec.Name,
			Priority:  jobItem.Spec.Priority,
			Status:    models.Running,
			CreatedAt: jobItem.Inserted,
			StartedAt: &now,
		}
		m.lock.Unlock()

		go m.submitJob(jobItem)
	}
}

func (m *Manager) submitJob(item *queue.JobItem) {
	jobName := item.Spec.Name + "-" + item.ID[:8]

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: m.namespace,
			Labels: map[string]string{
				"kubeserver/owner":    "true",
				"kubeserver/job-id":   item.ID,
				"kubeserver/priority": strconv.Itoa(item.Spec.Priority),
			},
		},
		Spec: batchv1.JobSpec{
			Template: item.Spec.Template,
		},
	}

	_, err := m.clientset.BatchV1().Jobs(m.namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("Failed to create job", slog.String("jobName", jobName), slog.String("error", err.Error()))

		m.lock.Lock()
		info := m.running[item.ID]
		info.Status = models.Failed
		info.ErrorMsg = err.Error()
		finished := time.Now()
		info.FinishedAt = &finished
		m.running[item.ID] = info
		m.lock.Unlock()
		return
	}

	m.logger.Info("Job created", slog.String("jobName", jobName))
}

func (m *Manager) RunningJobs() []models.JobInfo {
	m.lock.Lock()
	defer m.lock.Unlock()

	out := make([]models.JobInfo, 0, len(m.running))
	for _, job := range m.running {
		out = append(out, job)
	}
	return out
}

func (m *Manager) startJobInformer() {
	defer m.wg.Done()

	factory := informers.NewSharedInformerFactoryWithOptions(m.clientset, 0, informers.WithNamespace(m.namespace))
	jobInformer := factory.Batch().V1().Jobs().Informer()

	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			job := obj.(*batchv1.Job)
			m.logger.Debug("Job added in cluster", slog.String("name", job.Name))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			job := newObj.(*batchv1.Job)
			if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
				// m.logger.Info("Job finished in cluster", slog.String("name", job.Name), slog.Int32("succeeded", job.Status.Succeeded), slog.Int32("failed", job.Status.Failed))
				m.logger.Info("Job finished in cluster",
					slog.String("name", job.Name),
					slog.Int("succeeded", int(job.Status.Succeeded)),
					slog.Int("failed", int(job.Status.Failed)),
				)

				m.lock.Lock()
				defer m.lock.Unlock()
				for id, info := range m.running {
					if info.Name == job.Name || info.ID == job.Labels["kubeserver/job-id"] {
						fin := time.Now()
						info.FinishedAt = &fin
						if job.Status.Succeeded > 0 {
							info.Status = models.Completed
						} else {
							info.Status = models.Failed
						}
						delete(m.running, id)
						break
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			job := obj.(*batchv1.Job)
			m.logger.Debug("Job deleted in cluster", slog.String("name", job.Name))

			m.lock.Lock()
			defer m.lock.Unlock()
			for id, info := range m.running {
				if info.Name == job.Name || info.ID == job.Labels["kubeserver/job-id"] {
					delete(m.running, id)
					break
				}
			}
		},
	})

	stopInformerCh := make(chan struct{})
	factory.Start(stopInformerCh)
	factory.WaitForCacheSync(stopInformerCh)

	<-m.stopCh
	close(stopInformerCh)
}
