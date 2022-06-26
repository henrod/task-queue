package deprecated

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
	"time"

	"github.com/Henrod/task-queue/taskqueue"
)

var (
	baseOptions      *taskqueue.Options              // nolint:gochecknoglobals
	taskQueueMapping map[string]*taskqueue.TaskQueue // nolint:gochecknoglobals
	jobFuncMapping map[string]jobFunc                // nolint:gochecknoglobals
	redisClientMapping map[string]*redis.Client      // nolint:gochecknoglobals
)

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
func Process(queueKey string, job jobFunc, concurrency int) {
	var err error

	// Check if TaskQueue mapping already exists and create one if it does not
	if taskQueueMapping == nil {
		taskQueueMapping = map[string]*taskqueue.TaskQueue{}
	}

	// Check if TaskQueue object already exists and create one if it does not
	taskQueue, ok := taskQueueMapping[queueKey]
	if !ok {
		// Configure basic options
		ctx := context.Background()
		options := baseOptions.Copy()
		options.QueueKey = queueKey

		// Configure Redis client
		if redisClientMapping == nil {
			redisClientMapping = map[string]*redis.Client{}
		}
		redisClient, ok := redisClientMapping[queueKey]
		if !ok {
			redisClient = taskqueue.NewDefaultRedis(options)
			redisClientMapping[queueKey] = redisClient
		}

		// Create task queue
		taskQueue, err = taskqueue.NewTaskQueue(ctx, redisClient, options)
		if err != nil {
			log.Printf("failed to start taskQueue: %s", err)
			return
		}
		taskQueueMapping[queueKey] = taskQueue
	}

	// Check if JobFunc mapping already exists and create one if it does not
	if jobFuncMapping == nil {
		jobFuncMapping = map[string]jobFunc{}
	}

	// Bind job to queue through the mapping object
	jobFuncMapping[queueKey] = job
}

// Enqueue is not backward compatible yet since it retries by default.
func Enqueue(queueKey, class string, args interface{}) (string, error) {
	var err error
	ctx := context.Background()

	// Check if TaskQueue mapping already exists and create one if it does not
	if taskQueueMapping == nil {
		taskQueueMapping = map[string]*taskqueue.TaskQueue{}
	}

	// Check if TaskQueue object already exists and create one if it does not
	taskQueue, ok := taskQueueMapping[queueKey]
	if !ok {
		// Configure basic options
		options := baseOptions.Copy()
		options.QueueKey = queueKey

		// Configure Redis client
		if redisClientMapping == nil {
			redisClientMapping = map[string]*redis.Client{}
		}
		redisClient, ok := redisClientMapping[queueKey]
		if !ok {
			redisClient = taskqueue.NewDefaultRedis(options)
			redisClientMapping[queueKey] = redisClient
		}

		// Create task queue
		taskQueue, err = taskqueue.NewTaskQueue(
			ctx,
			redisClient,
			options,
		)
		if err != nil {
			log.Printf("failed to start taskQueue: %s", err)
			return "", fmt.Errorf("failed to start taskQueue: %w", err)
		}
		taskQueueMapping[queueKey] = taskQueue
	}

	// Produce message into specified queue
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

	for queueKey, taskQueue := range taskQueueMapping {

		if taskQueue == nil {
			log.Printf("nil task queue attached to queue %s", queueKey)
		}

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

			job, ok := jobFuncMapping[queueKey]
			if !ok {
				return fmt.Errorf("no job func mapped to queue %s", queueKey)
			}
			if job == nil {
				return fmt.Errorf("nil job func mapped to queue %s", queueKey)
			}
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
