package queue

import (
	"container/heap"
	"log/slog"
	"sync"
	"time"

	"github.com/abhi07kr/kubeserver/packages/models"
	"github.com/google/uuid"
)

type JobItem struct {
	ID       string
	Spec     models.JobSpec
	Inserted time.Time
	Index    int
}

type PriorityQueue struct {
	items  []*JobItem
	lock   sync.Mutex
	cond   *sync.Cond
	closed bool
	logger *slog.Logger
}

func NewPriorityQueue(logger *slog.Logger) *PriorityQueue {
	pq := &PriorityQueue{items: []*JobItem{}, logger: logger}
	pq.cond = sync.NewCond(&pq.lock)
	heap.Init(pq)
	return pq
}

func (pq *PriorityQueue) Len() int { return len(pq.items) }
func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Spec.Priority > pq.items[j].Spec.Priority
}
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index, pq.items[j].Index = i, j
}
func (pq *PriorityQueue) Push(x interface{}) { pq.items = append(pq.items, x.(*JobItem)) }
func (pq *PriorityQueue) Pop() interface{} {
	n := len(pq.items)
	itm := pq.items[n-1]
	pq.items = pq.items[:n-1]
	return itm
}

func (pq *PriorityQueue) Enqueue(spec models.JobSpec) (string, bool) {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	if pq.closed {
		return "", false
	}
	item := &JobItem{ID: uuid.NewString(), Spec: spec, Inserted: time.Now()}
	heap.Push(pq, item)
	pq.cond.Signal()
	if pq.logger != nil {
		pq.logger.Info("Job enqueued",
			slog.String("id", item.ID),
			slog.String("name", spec.Name),
			slog.Int("priority", spec.Priority),
		)
	}
	return item.ID, true
}

func (pq *PriorityQueue) DequeueBlocking() *JobItem {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	for pq.Len() == 0 && !pq.closed {
		pq.cond.Wait()
	}
	if pq.closed && pq.Len() == 0 {
		return nil
	}
	item := heap.Pop(pq).(*JobItem)
	if pq.logger != nil {
		pq.logger.Info("Job dequeued",
			slog.String("id", item.ID),
			slog.String("name", item.Spec.Name),
			slog.Int("priority", item.Spec.Priority),
		)
	}
	return item
}

func (pq *PriorityQueue) Close() {
	pq.lock.Lock()
	pq.closed = true
	pq.lock.Unlock()
	pq.cond.Broadcast()
}

func (pq *PriorityQueue) PendingJobs() []models.JobInfo {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	out := make([]models.JobInfo, 0, len(pq.items))
	for _, it := range pq.items {
		out = append(out, models.JobInfo{
			ID:        it.ID,
			Name:      it.Spec.Name,
			Priority:  it.Spec.Priority,
			Status:    models.Pending,
			CreatedAt: it.Inserted,
		})
	}
	return out
}

func (pq *PriorityQueue) List() []models.JobInfo {
	return pq.PendingJobs()
}
