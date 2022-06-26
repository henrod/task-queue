# Task Queue

[Sidekiq](http://sidekiq.org/) compatible background workers for [Golang](http://golang.org/).

* handles retries
* responds to Unix signals to safely wait for jobs to finish before exiting
* well tested

The application using this library will usually be divided into two independent parts: one set of producers (usually an API where the operations will be requested) and one set of consumers (usually background workers that will handle the heavy load).

## Consumer

The consumer app will be responsible for connecting to Redis and waiting for tasks to be queued by the producer.

Once the consumer detects a task, it will execute it using the provided callback function.

This can be done through the following source code example:

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

// Job function that will be binded to the first queue.
func myJobFunc(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
	// err := doSomethingWithYourMessage
	// if err != nil {
	// 	return fmt.Errorf("error while running my job func: %w", err)
	// }
	return nil // if no error happened during the job execution
}

// Job function that will be binded to the second queue.
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

	// Configure the first queue.
	firstQueueKey := "dummy-queue"
	firstWorkerID := "my-worker-ID"
	firstQueueOptions := &taskqueue.Options{
		QueueKey:         firstQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379", // address of the redis instance
		WorkerID:         firstWorkerID,
		MaxRetries:       3, // message will be discarded after 3 retries
		OperationTimeout: 2 * time.Minute,
	}
	// Instantiate a Task Queue object for the first queue.
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
	// Bind the first queue to the myJobFunc callback function.
	firstTaskQueue.Consume(
		ctx,
		func(ctx context.Context, taskID uuid.UUID, payload interface{}) error {
			firstLogger.Printf("consumed task %s from first queue with payload: %v\n", taskID, payload)
			return myJobFunc(ctx, taskID, payload)
		},
	)

	// Configure the second queue.
	secondQueueKey := "other-dummy-queue"
	secondWorkerID := "my-other-worker-ID"
	secondQueueOptions := &taskqueue.Options{
		QueueKey:         secondQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379", // address of the redis instance
		WorkerID:         secondWorkerID,
		MaxRetries:       -1, // unlimited max retries
		OperationTimeout: 1 * time.Minute,
	}
	// Instantiate a Task Queue object for the second queue.
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
	// Bind the second queue to the myOtherJobFunc callback function.
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

## Producer

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

	// Configure the first queue.
	firstQueueKey := "dummy-queue"
	firstQueueOptions := &taskqueue.Options{
		QueueKey:         firstQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
	}
	// Instantiate a Task Queue object for the first queue.
	firstTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(firstQueueOptions), firstQueueOptions)
	if err != nil {
		panic(err)
	}
	firstLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": firstQueueKey,
	})
	firstLogger.Info("producing task in first queue")
	// Produce a task to be consumed by the fist queue's consumer.
	firstQueueTaskID, err := firstTaskQueue.ProduceAt(ctx, &Payload{
		SomeKey: "some-value",
	}, time.Now())
	if err != nil {
		firstLogger.WithError(err).Error("failed to enqueue task to first queue")
	}
	firstLogger.Infof("enqueued task %s in first queue", firstQueueTaskID)

	// Configure the second queue.
	secondQueueKey := "other-dummy-queue"
	secondQueueOptions := &taskqueue.Options{
		QueueKey:         secondQueueKey,
		Namespace:        "my-namespace",
		StorageAddress:   "localhost:6379",
	}
	// Instantiate a Task Queue object for the second queue.
	secondTaskQueue, err := taskqueue.NewTaskQueue(ctx, taskqueue.NewDefaultRedis(secondQueueOptions), secondQueueOptions)
	if err != nil {
		panic(err)
	}
	secondLogger := logrus.New().WithFields(logrus.Fields{
		"operation": "consumer",
		"queueKey": secondQueueKey,
	})
	secondLogger.Info("producing task in second queue")
	// Produce a task to be consumed by the second queue's consumer.
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

## Testing Locally

You can quickly run the project locally using the example file stored in [examples/simple/main.go](./examples/simple/main.go).

In order to do so, all you need to do is:

- Run a local Redis instance:

```shell
$ redis-server
```

- Run the consumer by executing the example code passing 'consumer' as argument:

```shell
$ go run examples/simple/main.go consumer
```

- If no producer is running yet, you should just see a loop of "consuming task" messages:

```shell
INFO[0000] consuming task                                operation=consumer
INFO[0001] consuming task                                operation=consumer package=taskqueue
INFO[0002] consuming task                                operation=consumer package=taskqueue
INFO[0003] consuming task                                operation=consumer package=taskqueue
INFO[0004] consuming task                                operation=consumer package=taskqueue
INFO[0005] consuming task                                operation=consumer package=taskqueue
```

- Run the producer by executing the example code passing 'producer' as argument:

```shell
$ go run examples/simple/main.go producer
```

After running these 3 commands, you should start seeing messages like the ones below:

In the producer:

```shell
INFO[0001] producing task                                operation=producer
INFO[0001] enqueued task 67ff2e08-ccb7-4b94-ac9e-5be97268255c  operation=producer
INFO[0002] producing task                                operation=producer
INFO[0002] enqueued task 53679dc0-669f-4f97-89fc-431aa066c2df  operation=producer
INFO[0003] producing task                                operation=producer
INFO[0003] enqueued task 3d6b358e-3b43-4efe-8eba-b53cf1d2855a  operation=producer
INFO[0004] producing task                                operation=producer
INFO[0004] enqueued task d881ebc4-d8a8-4821-b89f-751d4f26c3ef  operation=producer
INFO[0005] producing task                                operation=producer
INFO[0005] enqueued task 502ea8be-90f2-41fa-92bb-8670ed652ea1  operation=producer
```

In the consumer:

```shell
INFO[0034] consuming task                                operation=consumer package=taskqueue
INFO[0034] consumed task 67ff2e08-ccb7-4b94-ac9e-5be97268255c: map[Body:0]  operation=consumer
INFO[0035] consuming task                                operation=consumer package=taskqueue
INFO[0035] consumed task 53679dc0-669f-4f97-89fc-431aa066c2df: map[Body:1]  operation=consumer
INFO[0036] consuming task                                operation=consumer package=taskqueue
INFO[0036] consumed task 3d6b358e-3b43-4efe-8eba-b53cf1d2855a: map[Body:2]  operation=consumer
INFO[0037] consuming task                                operation=consumer package=taskqueue
INFO[0037] consumed task d881ebc4-d8a8-4821-b89f-751d4f26c3ef: map[Body:3]  operation=consumer
INFO[0038] consuming task                                operation=consumer package=taskqueue
INFO[0038] consumed task 502ea8be-90f2-41fa-92bb-8670ed652ea1: map[Body:4]  operation=consumer
```

## To be implemented
* support custom middleware
* provide stats on what jobs are currently running

## Future implementation possibilities
* customize concurrency per queue

Initial development sponsored by [Customer.io](http://customer.io).
Implementation based on [jrallison/go-workers](https://github.com/jrallison/go-workers) and [topfreegames/go-workers](https://github.com/topfreegames/go-workers).
