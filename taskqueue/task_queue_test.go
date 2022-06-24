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

func TestTaskQueue_ProduceAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		now       = time.Now()
		mockRedis *taskqueue.MockRedis
		ctx       = context.Background()
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
		ctx       context.Context
		payload   interface{}
		executeAt time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		mock    func(*testing.T)
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
				ctx:       ctx,
				payload:   map[string]interface{}{"test": true},
				executeAt: now,
			},
			mock: func(t *testing.T) {
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(redis.NewStringCmd(ctx))

				mockRedis.EXPECT().
					ZAdd(ctx, "taskqueue:test-namespace:tasks:test-queue", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, redisZ *redis.Z) *redis.IntCmd {
						if redisZ.Score != float64(now.Unix()) {
							t.Errorf("TaskQueue.ProduceAt() expected = %v, wantErr %v", redisZ.Score, float64(now.Unix()))
						}

						payload := redisZ.Member.(*taskqueue.Task).Payload
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
			name: "test_failed_redis_err",
			fields: fields{
				queueKey:         "test-queue",
				namespace:        "test-namespace",
				storageAddress:   "",
				workerID:         "worker-0",
				maxRetries:       0,
				operationTimeout: time.Minute,
			},
			args: args{
				ctx:       ctx,
				payload:   map[string]interface{}{"test": false},
				executeAt: now,
			},
			mock: func(t *testing.T) {
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(redis.NewStringCmd(ctx))

				cmd := redis.NewIntCmd(ctx)
				cmd.SetErr(errors.New("some error"))
				mockRedis.EXPECT().
					ZAdd(ctx, "taskqueue:test-namespace:tasks:test-queue", gomock.Any()).
					Return(cmd)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis = taskqueue.NewMockRedis(ctrl)

			tt.mock(t)

			taskQueue, err := taskqueue.NewTaskQueue(
				tt.args.ctx,
				mockRedis,
				&taskqueue.Options{
					QueueKey:         tt.fields.queueKey,
					Namespace:        tt.fields.namespace,
					StorageAddress:   tt.fields.storageAddress,
					WorkerID:         tt.fields.workerID,
					MaxRetries:       tt.fields.maxRetries,
					OperationTimeout: tt.fields.operationTimeout,
				},
			)
			if err != nil {
				t.Errorf("taskqueue.NewTaskQueue error = %v", err)
				return
			}

			got, err := taskQueue.ProduceAt(tt.args.ctx, tt.args.payload, tt.args.executeAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskQueue.ProduceAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == uuid.Nil && !tt.wantErr {
				t.Errorf("TaskQueue.ProduceAt() = %v, want not nil", got)
			}
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

func TestNewTaskQueue(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		now       = time.Now()
		mockRedis *taskqueue.MockRedis
		ctx       = context.Background()
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
		ctx       context.Context
		payload   interface{}
		executeAt time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		mock    func(*testing.T)
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
				ctx:       ctx,
				payload:   map[string]interface{}{"test": false},
				executeAt: now,
			},
			mock: func(t *testing.T) {
				cmd := redis.NewStringCmd(ctx)
				cmd.SetErr(errors.New("some error"))
				mockRedis.EXPECT().
					ScriptLoad(ctx, readScriptFile(t)).
					Return(cmd)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &taskqueue.Options{
				QueueKey:         tt.fields.queueKey,
				Namespace:        tt.fields.namespace,
				StorageAddress:   tt.fields.storageAddress,
				WorkerID:         tt.fields.workerID,
				MaxRetries:       tt.fields.maxRetries,
				OperationTimeout: tt.fields.operationTimeout,
			}
			mockRedis = taskqueue.NewMockRedis(ctrl)
			tt.mock(t)
			got, err := taskqueue.NewTaskQueue(tt.args.ctx, mockRedis, options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQueue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("NewTaskQueue() expect TaskQueue not nil")
				return
			}
		})
	}
}
