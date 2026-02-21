package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-worker-go/activities"
	"temporal-worker-go/workflows"
)

func main() {
	// Read configuration from environment
	temporalAddress := getEnv("TEMPORAL_ADDRESS", "localhost:7233")
	temporalNamespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TEMPORAL_TASK_QUEUE", "go-task-queue")

	log.Printf("Starting Go worker...")
	log.Printf("  Temporal:  %s", temporalAddress)
	log.Printf("  Namespace: %s", temporalNamespace)
	log.Printf("  TaskQueue: %s", taskQueue)

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: temporalNamespace,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.EventProcessingWorkflow)
	w.RegisterWorkflow(workflows.DummyWaitWorkflow)
	w.RegisterWorkflow(workflows.HumanApprovalWorkflow)
	w.RegisterWorkflow(workflows.SagaWorkflow)
	w.RegisterWorkflow(workflows.ChildFanoutWorkflow)
	w.RegisterWorkflow(workflows.ChildProcessItemWorkflow)

	// Register activities
	w.RegisterActivity(activities.ValidateEvent)
	w.RegisterActivity(activities.ProcessEvent)
	w.RegisterActivity(activities.NotifyEventProcessed)
	w.RegisterActivity(activities.SendApprovalRequest)
	w.RegisterActivity(activities.NotifyApprovalResult)
	w.RegisterActivity(activities.ReserveInventory)
	w.RegisterActivity(activities.ChargePayment)
	w.RegisterActivity(activities.ShipOrder)
	w.RegisterActivity(activities.CancelReservation)
	w.RegisterActivity(activities.RefundPayment)
	w.RegisterActivity(activities.ProcessItem)

	log.Printf("Worker started. Listening on task queue: %s", taskQueue)

	// Start listening for tasks — this blocks
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
