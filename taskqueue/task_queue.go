package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type TaskQueue struct {
	redis             *redis.Client
	taskQueueKey      string
	inProgressTaskKey string
	consumeScriptSha  string
	maxRetries        int
	operationTimeout  time.Duration
}

type Task struct {
	ID         uuid.UUID
	Payload    interface{}
	RetryCount int
	Wait       time.Duration
}

const consumeScriptPath = "./taskqueue/consume.lua"

func NewTaskQueue(ctx context.Context, options *Options) (*TaskQueue, error) {
	options.setDefaults()
	redisClient := redis.NewClient(&redis.Options{Addr: options.Address})

	consumeScriptBytes, err := os.ReadFile(consumeScriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read consume script file: %w", err)
	}

	consumeScriptSHA, err := redisClient.ScriptLoad(ctx, string(consumeScriptBytes)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load consume script file into redis: %w", err)
	}

	taskQueue := &TaskQueue{
		redis:             redisClient,
		taskQueueKey:      fmt.Sprintf("taskqueue:%s:tasks", options.Namespace),
		inProgressTaskKey: fmt.Sprintf("taskqueue:%s:workers:%s:tasks", options.Namespace, options.WorkerID),
		consumeScriptSha:  consumeScriptSHA,
		maxRetries:        options.MaxRetries,
		operationTimeout:  options.OperationTimeout,
	}

	return taskQueue, nil
}

func (t *TaskQueue) ProduceAt(ctx context.Context, payload interface{}, executeAt time.Time) (uuid.UUID, error) {
	logger := newLogger()

	task := &Task{
		ID:         uuid.New(),
		Payload:    payload,
		RetryCount: 0,
		Wait:       0,
	}

	logger.Debugf("producing task %s %v", task.ID, task.Payload)
	err := t.produceAt(ctx, task, executeAt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to produce task: %w", err)
	}

	return task.ID, nil
}

func (t *TaskQueue) Consume(
	ctx context.Context,
	consume func(context.Context, uuid.UUID, interface{}) error,
) error {
	logger := newLogger()

	now := time.Now()
	keys := []string{
		t.taskQueueKey,
		t.inProgressTaskKey,
	}
	args := []interface{}{
		fmt.Sprintf("%d", now.Unix()),
	}

	logger.Debugf("fetching task to execute")
	taskStr, err := t.redis.EvalSha(ctx, t.consumeScriptSha, keys, args...).Result()
	if err != nil {
		return fmt.Errorf("failed to execute consume script: %w", err)
	}

	if taskStr == StatusOK {
		logger.Debug("no tasks to execute")
		return nil
	}

	task := new(Task)
	err = json.Unmarshal([]byte(taskStr.(string)), task)
	if err != nil {
		return fmt.Errorf("failed to unmarshal task and permanentely lost it: %w", err)
	}

	logger = withTaskLabels(logger, task)
	defer t.removeInProgressTask(ctx, task)

	logger.Debug("consuming task")
	err = consume(ctx, task.ID, task.Payload)
	if err != nil {
		logger.WithError(err).Debug("failed to consume, retrying after backoff")
		retryErr := t.produceRetry(ctx, task)
		if retryErr != nil {
			return ErrTaskLost(
				task.ID,
				fmt.Errorf("failed to run consume func and failed to enqueue it for retry: %w", retryErr),
			)
		}

		return fmt.Errorf("failed to run consume func: %w", err)
	}

	logger.Debug("successfully consumed task")

	return nil
}

func (t *TaskQueue) produceAt(
	ctx context.Context,
	task *Task,
	executeAt time.Time,
) error {
	bTask, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to json marshal message: %w", err)
	}

	if err := t.redis.ZAdd(ctx, t.taskQueueKey, &redis.Z{
		Score:  float64(executeAt.Unix()),
		Member: string(bTask),
	}).Err(); err != nil {
		return fmt.Errorf("failed to zadd message: %w", err)
	}

	return nil
}

func (t *TaskQueue) produceRetry(ctx context.Context, task *Task) error {
	now := time.Now()

	if t.maxRetries >= 0 && task.RetryCount >= t.maxRetries {
		return fmt.Errorf("task reached max retries: %s, %d", task.ID, task.RetryCount)
	}

	wait := time.Second
	if task.Wait > 0 {
		wait = task.Wait * 2
	}

	retryTask := &Task{
		ID:         task.ID,
		Payload:    task.Payload,
		RetryCount: task.RetryCount + 1,
		Wait:       wait,
	}

	executeAt := now.Add(wait)

	err := t.produceAt(ctx, retryTask, executeAt)
	if err != nil {
		return fmt.Errorf("failed to produce retry task: %w", err)
	}

	return nil
}

func (t *TaskQueue) removeInProgressTask(ctx context.Context, task *Task) {
	logger := newLogger()
	logger = withTaskLabels(logger, task)

	err := t.redis.Del(ctx, t.inProgressTaskKey).Err()
	if err != nil {
		logger.
			WithError(err).
			Warn("failed to delete worker in progress task, might duplicate if worker restart now")
	}
}
