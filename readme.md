Kubernetes Job Server
A small Go HTTP server to submit and manage Kubernetes Jobs with priority queuing and concurrency limits. It talks to Kubernetes using client-go, keeps track of running and pending jobs, and exposes a simple REST API.

What it does
Accepts job submissions with a priority level

Uses a priority queue to schedule jobs

Runs jobs concurrently (configurable max concurrency)

Tracks running jobs by watching the cluster

Lets you list pending and running jobs via API

Gracefully shuts down workers and HTTP server

Requirements
Go 1.21 or newer


Access to a Kubernetes cluster and kubeconfig (or run in-cluster)


To Run It :
git clone https://github.com/abhi07kr/kubeserver.git
cd kubeserver


go run ./cli --kubeconfig ~/.kube/config --port 8080 --max-concurrency 3


NOTE: Task 2 is also in this repository. (Task2.md)



