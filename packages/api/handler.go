package api

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/abhi07kr/kubeserver/packages/models"
	"github.com/abhi07kr/kubeserver/packages/queue"
	"github.com/abhi07kr/kubeserver/packages/worker"

	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
)

type Handler struct {
	pq     *queue.PriorityQueue
	mgr    *worker.Manager
	client *kubernetes.Clientset
	ns     string
	logger *slog.Logger
}

func NewHandler(pq *queue.PriorityQueue, mgr *worker.Manager, client *kubernetes.Clientset, namespace string, logger *slog.Logger) *Handler {
	return &Handler{
		pq:     pq,
		mgr:    mgr,
		client: client,
		ns:     namespace,
		logger: logger,
	}
}

func (h *Handler) Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/jobs", h.SubmitJob).Methods("POST")
	r.HandleFunc("/jobs/pending", h.GetPending).Methods("GET")
	r.HandleFunc("/jobs/running", h.GetRunning).Methods("GET")
	return r
}

func (h *Handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var spec models.JobSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		h.logger.Error("invalid job payload", slog.Any("error", err))
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id, ok := h.pq.Enqueue(spec)
	if !ok {
		h.logger.Error("queue closed, cannot accept job")
		http.Error(w, "service shutting down", http.StatusServiceUnavailable)
		return
	}
	resp := map[string]string{"id": id}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetPending(w http.ResponseWriter, r *http.Request) {
	jobs := h.pq.List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobs)
}

func (h *Handler) GetRunning(w http.ResponseWriter, r *http.Request) {
	jobs := h.mgr.RunningJobs()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobs)
}
