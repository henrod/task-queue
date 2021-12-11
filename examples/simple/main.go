package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Henrod/task-queue/taskqueue"
	"github.com/google/uuid"
)

type Payload struct {
	Body string
}

var wg sync.WaitGroup

func produce(ctx context.Context, taskQueue *taskqueue.TaskQueue) {
	defer wg.Done()

	var (
		now    = time.Now()
		ticker = time.NewTicker(time.Second)
		logger = logrus.New().WithFields(logrus.Fields{
			"operation": "producer",
		})
	)

	for {
		select {
		case <-ticker.C:
			logger.Info("producing task")
			taskID, err := taskQueue.ProduceAt(ctx, &Payload{Body: "foo"}, now.Add(10*time.Second))
			if err != nil {
				logger.WithError(err).Error("failed to enqueue task")
				break
			}

			logger.Infof("enqueued task %s", taskID)

		case <-ctx.Done():
			logger.Info("stopping")
			return
		}
	}
}

func consume(ctx context.Context, taskQueue *taskqueue.TaskQueue) {
	defer wg.Done()

	var (
		ticker = time.NewTicker(time.Second)
		logger = logrus.New().WithFields(logrus.Fields{
			"operation": "consumer",
		})
	)

	for {
		select {
		case <-ticker.C:
			logger.Info("consuming task")
			if err := taskQueue.Consume(
				context.Background(),
				func(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
					logger.Printf("consuming task %s: %v\n", taskID, payload)
					time.Sleep(10 * time.Second)
					logger.Printf("consumed task %s: %v\n", taskID, payload)
					return errors.New("some error")
				},
			); err != nil {
				logger.WithError(err).Error("failed to consume task")
			}
		case <-ctx.Done():
			logger.Info("stopping")
			return
		}
	}

}

func handleStop(cancel context.CancelFunc) {
	logger := logrus.New()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	logger.Info("received termination signal, waiting for operations to finish")
	cancel()
}

func run() error {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		logger      = logrus.New()
	)

	go handleStop(cancel)

	taskQueue, err := taskqueue.NewTaskQueue(ctx, &taskqueue.Options{
		Namespace:        "simple",
		Address:          "localhost:6379",
		WorkerID:         "worker1",
		MaxRetries:       -1,
		OperationTimeout: time.Minute,
	})
	if err != nil {
		return fmt.Errorf("failed to start task queue: %w", err)
	}

	//wg.Add(1)
	//go produce(ctx, taskQueue)

	wg.Add(1)
	go consume(ctx, taskQueue)

	wg.Wait()
	logger.Info("done")

	return nil
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	if err := run(); err != nil {
		panic(err)
	}
}
