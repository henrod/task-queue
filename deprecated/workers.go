package deprecated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Henrod/task-queue/taskqueue"
	"github.com/google/uuid"
	"github.com/topfreegames/go-workers"
)

var (
	baseOptions      *taskqueue.Options              // nolint:gochecknoglobals
	taskQueueMapping map[string]*taskqueue.TaskQueue // nolint:gochecknoglobals
)

var ErrQueueNotFound = errors.New("queue not registered to Process yet")

type jobFunc func(message *workers.Msg)

func Configure(optionsMap map[string]string) {
	baseOptions = &taskqueue.Options{
		QueueKey:         "",
		Namespace:        optionsMap["namespace"],
		StorageAddress:   optionsMap["server"],
		WorkerID:         optionsMap["process"],
		MaxRetries:       -1,
		OperationTimeout: time.Minute,
	}
}

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

	taskQueueMapping[queue] = taskQueue

	go taskQueue.Consume(ctx, func(ctx context.Context, taskID uuid.UUID, payload interface{}) (err error) {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		msg, err := workers.NewMsg(string(payloadBytes))
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

		job(msg)

		return nil
	})
}

// Enqueue is not backward compatible yet since it retries by default.
func Enqueue(queue, class string, args interface{}) (string, error) {
	taskQueue, ok := taskQueueMapping[queue]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrQueueNotFound, queue)
	}

	taskID, err := taskQueue.ProduceAt(
		context.Background(),
		args,
		time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue: %w", err)
	}

	return taskID.String(), nil
}

// EnqueueWithOptions is not backward compatible yet since it doesn't accept another Redis connection.
func EnqueueWithOptions(queue, class string, args interface{}, opts workers.EnqueueOptions) (string, error) {
	return Enqueue(queue, class, args)
}
