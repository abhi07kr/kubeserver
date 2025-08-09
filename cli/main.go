package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhi07kr/kubeserver/packages/api"
	"github.com/abhi07kr/kubeserver/packages/k8s"
	"github.com/abhi07kr/kubeserver/packages/queue"
	"github.com/abhi07kr/kubeserver/packages/worker"
)

func main() {
	kubeconfig := flag.String("kubeconfig", os.Getenv("HOME")+"/.kube/config", "Path to kubeconfig")
	port := flag.Int("port", 8080, "HTTP server port")
	maxConcurrency := flag.Int("max-concurrency", 3, "Max concurrent job submissions")
	namespace := flag.String("namespace", "default", "Kubernetes namespace")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	logger.Info("Starting server",
		"port", *port,
		"kubeconfig", *kubeconfig,
		"maxConcurrency", *maxConcurrency,
		"namespace", *namespace,
	)

	clientset, err := k8s.NewKubeClient(*kubeconfig)
	if err != nil {
		logger.Error("failed to create kube client", "error", err)
		os.Exit(1)
	}

	pq := queue.NewPriorityQueue(logger)

	mgr := worker.NewManager(*maxConcurrency, clientset, pq, logger, *namespace)

	// mgr.Start() removed because NewManager already calls Start()

	handler := api.NewHandler(pq, mgr, clientset, *namespace, logger)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: handler.Router(),
	}

	go func() {
		logger.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("http shutdown error", "error", err)
	}

	mgr.Stop()

	logger.Info("shutdown complete")
}
