package deprecated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"time"

	"github.com/Henrod/task-queue/taskqueue"
)

var (
	baseOptions      *taskqueue.Options              // nolint:gochecknoglobals
	taskQueueMapping map[string]*taskqueue.TaskQueue // nolint:gochecknoglobals
	jobFuncMapping map[string]jobFunc // nolint:gochecknoglobals
)

var ErrQueueNotFound = errors.New("queue not registered to Process yet")

type jobFunc func(message *Msg)
var Config *config

// Configure creates a base Options object based on the information provided in the options map.
func Configure(optionsMap map[string]string) {
	baseOptions = &taskqueue.Options{
		QueueKey:         "",
		Namespace:        optionsMap["namespace"],
		StorageAddress:   optionsMap["server"],
		WorkerID:         optionsMap["process"],
		MaxRetries:       -1,
		OperationTimeout: time.Minute,
	}

	Config = &config{
		Pool: &Pool{
			Get: func() Conn {
				return Conn{
					Err: func() error {
						return nil
					},
					Close: func() {},
				}
			},
		},
	}
}

// Process binds a job function to a specific queue.
func Process(queue string, job jobFunc, concurrency int) {
	ctx := context.Background()
	options := baseOptions.Copy()
	options.QueueKey = queue
	redisClient := taskqueue.NewDefaultRedis(options)

	taskQueue, err := taskqueue.NewTaskQueue(
		ctx,
		redisClient,
		options,
	)
	if err != nil {
		log.Printf("failed to start taskQueue: %s", err)

		return
	}
	if taskQueueMapping == nil {
		taskQueueMapping = map[string]*taskqueue.TaskQueue{}
	}
	taskQueueMapping[queue] = taskQueue
	if jobFuncMapping == nil {
		jobFuncMapping = map[string]jobFunc{}
	}
	jobFuncMapping[queue] = job
}

// Enqueue is not backward compatible yet since it retries by default.
func Enqueue(queue, class string, args interface{}) (string, error) {
	ctx := context.Background()

	taskQueue, ok := taskQueueMapping[queue]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrQueueNotFound, queue)
	}

	taskID, err := taskQueue.ProduceAt(
		ctx,
		args,
		time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue: %w", err)
	}

	return taskID.String(), nil
}

// EnqueueWithOptions is not backward compatible yet since it doesn't accept another Redis connection.
func EnqueueWithOptions(queue, class string, args interface{}, opts EnqueueOptions) (string, error) {
	return Enqueue(queue, class, args)
}

// Run loops over all the queues configured and starts consuming each one of them using the respective jobs.
func Run() {
	ctx := context.Background()

	for queueName, taskQueue := range taskQueueMapping {
		go taskQueue.Consume(ctx, func(ctx context.Context, taskID uuid.UUID, payload interface{}) (err error) {
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}

			msg, err := NewMsg(string(payloadBytes))
			if err != nil {
				return fmt.Errorf("failed to build *workers.Msg: %w", err)
			}

			defer func() {
				if r := recover(); r != nil {
					if panicErr, ok := r.(error); ok {
						err = panicErr
					}
				}
			}()

			job := jobFuncMapping[queueName]
			job(msg)

			return nil
		})
	}
}

// Quit loops over all the queues configured and removes their references so the GC can free the respective memory.
func Quit() {
	for queueName, _ := range taskQueueMapping {
		taskQueueMapping[queueName] = nil
	}
}
