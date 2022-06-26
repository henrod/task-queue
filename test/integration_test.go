package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Henrod/task-queue/taskqueue"
	"github.com/google/uuid"
)

func TestTaskQueueMainFlow(t *testing.T) {
	options := &taskqueue.Options{
		QueueKey:         "integration-test",
		Namespace:        "simple",
		StorageAddress:   "localhost:6379",
		WorkerID:         "worker1",
		MaxRetries:       0,
		OperationTimeout: time.Minute,
	}

	ctx, cancel := context.WithCancel(context.Background())
	taskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(options), options)
	if err != nil {
		panic(err)
	}

	StartProducer(ctx, taskQueue)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	StartConsumer(ctx, cancel, taskQueue, waitGroup)
}

type Payload struct {
	Body string
}

func StartProducer(ctx context.Context, taskQueue *taskqueue.TaskQueue) error {
	id := 1234
	_, err := taskQueue.ProduceAt(ctx, &Payload{Body: fmt.Sprintf("%d", id)}, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func StartConsumer(ctx context.Context, cancel context.CancelFunc, taskQueue *taskqueue.TaskQueue, waitGroup sync.WaitGroup) error {
	taskQueue.Consume(
		ctx,
		func(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
			waitGroup.Done()
			waitGroup.Wait()
			cancel()
			return nil
		},
	)
	return nil
}
