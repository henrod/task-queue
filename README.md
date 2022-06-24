[Sidekiq](http://sidekiq.org/) compatible
background workers in [golang](http://golang.org/).

* handles retries
* responds to Unix signals to safely wait for jobs to finish before exiting
* well tested

The application using this library will usually be divided into two independent parts: one set of producers (usually an API where the operations will be requested) and one set of consumers (usually background workers).

The consumer app will be responsible for connecting to Redis and waiting for operations to be queued by the producer. This can be done through the following source code example:

```go
package main

import (
	"context"
	// "fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/Henrod/task-queue/taskqueue"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func myJobFunc(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
	// err := doSomethingWithYourMessage
	// if err != nil {
	// 	return fmt.Errorf("error while running my job func: %w", err)
	// }
	return nil // if no error happened during the job execution
}

func myOtherJobFunc(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
	// err := doSomethingWithYourMessage
	// if err != nil {
	// 	return fmt.Errorf("error while running my job func: %w", err)
	// }
	return nil // if no error happened during the job execution
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go handleStop(cancel)

	firstQueueKey := "dummy-queue"
	firstWorkerID := "my-worker-ID"
	firstQueueOptions := &taskqueue.Options{
		QueueKey:         firstQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
		WorkerID:         firstWorkerID,
		MaxRetries:       3, // message will be discarded after 3 retries
		OperationTimeout: 2 * time.Minute,
	}
	firstTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(firstQueueOptions), firstQueueOptions)
	if err != nil {
		panic(err)
	}
	firstLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": firstQueueKey,
		"workerID": firstWorkerID,
	})
	firstLogger.Info("consuming tasks from first queue")
	firstTaskQueue.Consume(
		ctx,
		func(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
			firstLogger.Printf("consumed task %s from first queue with payload: %v\n", taskID, payload)
			return myJobFunc(ctx, taskID, payload)
		},
	)

	secondQueueKey := "other-dummy-queue"
	secondWorkerID := "my-other-worker-ID"
	secondQueueOptions := &taskqueue.Options{
		QueueKey:         secondQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
		WorkerID:         secondWorkerID,
		MaxRetries:       -1, // unlimited max retries
		OperationTimeout: 1 * time.Minute,
	}
	secondTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(secondQueueOptions), secondQueueOptions)
	if err != nil {
		panic(err)
	}
	secondLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": secondQueueKey,
		"worker": secondWorkerID,
	})
	secondLogger.Info("consuming tasks from second queue")
	secondTaskQueue.Consume(
		ctx,
		func(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
			secondLogger.Printf("consumed task %s from second queue with payload: %v\n", taskID, payload)
			return myJobFunc(ctx, taskID, payload)
		},
	)
}

func handleStop(cancel context.CancelFunc) {
	logger := logrus.New()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	logger.Info("received termination signal, waiting for operations to finish")
	cancel()
}
```

The producer app will be responsible for connecting to Redis and enqueueing operations to be processed by the consumer. This can be done through the following source code example:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/Henrod/task-queue/taskqueue"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Payload struct {
	SomeKey string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go handleStop(cancel)

	firstQueueKey := "dummy-queue"
	firstQueueOptions := &taskqueue.Options{
		QueueKey:         firstQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
	}
	firstTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(firstQueueOptions), firstQueueOptions)
	if err != nil {
		panic(err)
	}
	firstLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": firstQueueKey,
	})
	firstLogger.Info("producing task in first queue")
	firstQueueTaskID, err := firstTaskQueue.ProduceAt(ctx, &Payload{
		SomeKey: "some-value",
	}, time.Now())
	if err != nil {
		firstLogger.WithError(err).Error("failed to enqueue task to first queue")
	}
	firstLogger.Infof("enqueued task %s in first queue", firstQueueTaskID)

	secondQueueKey := "other-dummy-queue"
	secondQueueOptions := &taskqueue.Options{
		QueueKey:         secondQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
	}
	secondTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(secondQueueOptions), secondQueueOptions)
	if err != nil {
		panic(err)
	}
	secondLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": secondQueueKey,
	})
	secondLogger.Info("producing task in second queue")
	secondQueueTaskID, err := secondTaskQueue.ProduceAt(ctx, &Payload{
		SomeKey: "some-value",
	}, time.Now())
	if err != nil {
		secondLogger.WithError(err).Error("failed to enqueue task to second queue")
	}
	secondLogger.Infof("enqueued task %s in second queue", secondQueueTaskID)
}

func handleStop(cancel context.CancelFunc) {
	logger := logrus.New()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	logger.Info("received termination signal, waiting for operations to finish")
	cancel()
}
```

To be implemented:
* supports custom middleware
* provides stats on what jobs are currently running

Future implementation possibilities:
* customize concurrency per queue

Initial development sponsored by [Customer.io](http://customer.io)
