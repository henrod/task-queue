//go:build unit
// +build unit

package taskqueue_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Henrod/task-queue/taskqueue"
	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

var errSomeError = errors.New("some error")

func TestNewTaskQueue(t *testing.T) { //nolint:funlen
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	var (
		now = time.Now()
		ctx = context.Background()
	)

	type fields struct {
		queueKey         string
		namespace        string
		storageAddress   string
		workerID         string
		maxRetries       int
		operationTimeout time.Duration
	}

	type args struct {
		payload   interface{}
		executeAt time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		mock    func(*testing.T, *taskqueue.MockRedis)
		wantErr bool
	}{
		{
			name: "test_failed_instantiate_task_queue",
			fields: fields{
				queueKey:         "test-queue",
				namespace:        "test-namespace",
				storageAddress:   "",
				workerID:         "worker-0",
				maxRetries:       0,
				operationTimeout: time.Minute,
			},
			args: args{
				payload:   map[string]interface{}{"test": false},
				executeAt: now,
			},
			mock: func(t *testing.T, mockRedis *taskqueue.MockRedis) {
				cmd := redis.NewStringCmd(ctx)
				cmd.SetErr(errSomeError)
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(cmd)
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options := &taskqueue.Options{
				QueueKey:         test.fields.queueKey,
				Namespace:        test.fields.namespace,
				StorageAddress:   test.fields.storageAddress,
				WorkerID:         test.fields.workerID,
				MaxRetries:       test.fields.maxRetries,
				OperationTimeout: test.fields.operationTimeout,
			}

			mockRedis := taskqueue.NewMockRedis(ctrl)

			test.mock(t, mockRedis)

			got, err := taskqueue.NewTaskQueue(ctx, mockRedis, options)
			if (err != nil) != test.wantErr {
				t.Errorf("NewTaskQueue() error = %v, wantErr %v", err, test.wantErr)

				return
			}
			if !test.wantErr && got == nil {
				t.Errorf("NewTaskQueue() expect TaskQueue not nil")

				return
			}
		})
	}
}

func TestTaskQueue_ProduceAt(t *testing.T) { //nolint:funlen
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	var (
		now = time.Now()
		ctx = context.Background()
	)

	type fields struct {
		queueKey         string
		namespace        string
		storageAddress   string
		workerID         string
		maxRetries       int
		operationTimeout time.Duration
	}

	type args struct {
		payload   interface{}
		executeAt time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		mock    func(*testing.T, *taskqueue.MockRedis)
		wantErr bool
	}{
		{
			name: "test_success",
			fields: fields{
				queueKey:         "test-queue",
				namespace:        "test-namespace",
				storageAddress:   "",
				workerID:         "worker-0",
				maxRetries:       0,
				operationTimeout: time.Minute,
			},
			args: args{
				payload:   map[string]interface{}{"test": true},
				executeAt: now,
			},
			mock: func(t *testing.T, mockRedis *taskqueue.MockRedis) {
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(redis.NewStringCmd(ctx))

				mockRedis.EXPECT().
					ZAdd(ctx, "taskqueue:test-namespace:tasks:test-queue", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, redisZ *redis.Z) *redis.IntCmd {
						if redisZ.Score != float64(now.Unix()) {
							t.Errorf("TaskQueue.ProduceAt() expected = %v, wantErr %v", redisZ.Score, float64(now.Unix()))
						}

						payload := redisZ.Member.(*taskqueue.Task).Payload //nolint:forcetypeassert
						expectedPayload := map[string]interface{}{"test": true}
						if !reflect.DeepEqual(payload, expectedPayload) {
							t.Errorf("TaskQueue.ProduceAt() expectedPayload = %v, wantPayload %v", expectedPayload, payload)
						}

						return redis.NewIntCmd(ctx)
					})
			},
			wantErr: false,
		},
		{
			name: "test_zadd_error",
			fields: fields{
				queueKey:         "test-queue",
				namespace:        "test-namespace",
				storageAddress:   "",
				workerID:         "worker-0",
				maxRetries:       0,
				operationTimeout: time.Minute,
			},
			args: args{
				payload:   map[string]interface{}{"test": false},
				executeAt: now,
			},
			mock: func(t *testing.T, mockRedis *taskqueue.MockRedis) {
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(redis.NewStringCmd(ctx))

				cmd := redis.NewIntCmd(ctx)
				cmd.SetErr(errSomeError)
				mockRedis.EXPECT().
					ZAdd(ctx, "taskqueue:test-namespace:tasks:test-queue", gomock.Any()).
					Return(cmd)
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mockRedis := taskqueue.NewMockRedis(ctrl)

			test.mock(t, mockRedis)

			taskQueue, err := taskqueue.NewTaskQueue(
				ctx,
				mockRedis,
				&taskqueue.Options{
					QueueKey:         test.fields.queueKey,
					Namespace:        test.fields.namespace,
					StorageAddress:   test.fields.storageAddress,
					WorkerID:         test.fields.workerID,
					MaxRetries:       test.fields.maxRetries,
					OperationTimeout: test.fields.operationTimeout,
				},
			)
			if err != nil {
				t.Errorf("taskqueue.NewTaskQueue error = %v", err)

				return
			}

			got, err := taskQueue.ProduceAt(ctx, test.args.payload, test.args.executeAt)
			if (err != nil) != test.wantErr {
				t.Errorf("TaskQueue.ProduceAt() error = %v, wantErr %v", err, test.wantErr)

				return
			}
			if got == uuid.Nil && !test.wantErr {
				t.Errorf("TaskQueue.ProduceAt() = %v, want not nil", got)
			}
		})
	}
}

func TestTaskQueue_Consume(t *testing.T) { //nolint:funlen
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	type fields struct {
		queueKey         string
		namespace        string
		storageAddress   string
		workerID         string
		maxRetries       int
		operationTimeout time.Duration
	}

	type args struct {
		consume func(context.CancelFunc) taskqueue.ConsumeFunc
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		mock    func(*testing.T, context.Context, *taskqueue.MockRedis)
		wantErr bool
	}{
		{
			name: "test_success",
			fields: fields{
				queueKey:         "test-queue",
				namespace:        "test-namespace",
				storageAddress:   "",
				workerID:         "worker-0",
				maxRetries:       0,
				operationTimeout: time.Minute,
			},
			args: args{
				consume: func(cancelFunc context.CancelFunc) taskqueue.ConsumeFunc {
					return func(context.Context, uuid.UUID, interface{}) error {
						cancelFunc()

						return nil
					}
				},
			},
			mock: func(t *testing.T, ctx context.Context, mockRedis *taskqueue.MockRedis) {
				taskBytes, _ := new(taskqueue.Task).MarshalBinary()
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(redis.NewStringCmd(ctx))

				mockRedis.EXPECT().
					EvalSha(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(redis.NewCmdResult(string(taskBytes), nil))

				mockRedis.EXPECT().
					Del(ctx, gomock.Any()).
					Return(redis.NewIntCmd(ctx))
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mockRedis := taskqueue.NewMockRedis(ctrl)
			ctx, cancel := context.WithCancel(context.Background())

			test.mock(t, ctx, mockRedis)

			taskQueue, err := taskqueue.NewTaskQueue(
				ctx,
				mockRedis,
				&taskqueue.Options{
					QueueKey:         test.fields.queueKey,
					Namespace:        test.fields.namespace,
					StorageAddress:   test.fields.storageAddress,
					WorkerID:         test.fields.workerID,
					MaxRetries:       test.fields.maxRetries,
					OperationTimeout: test.fields.operationTimeout,
				},
			)
			if err != nil {
				t.Errorf("taskqueue.NewTaskQueue error = %v", err)
			}

			taskQueue.Consume(ctx, test.args.consume(cancel))
		})
	}
}

func readScriptFile(t *testing.T) string {
	t.Helper()

	consumeScriptBytes, err := os.ReadFile("consume.lua")
	if err != nil {
		t.Fatalf("failed to read consume.lua file: %s", err)
	}

	return string(consumeScriptBytes)
}
