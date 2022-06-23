package taskqueue_test

import (
	"context"
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

	now := time.Now()
	var mockRedis *taskqueue.MockRedis

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
		mock    func()
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
				ctx:       context.Background(),
				payload:   map[string]interface{}{"test": true},
				executeAt: now,
			},
			mock: func() {
				mockRedis.EXPECT().ScriptLoad(gomock.Any(), gomock.Any()).Return(redis.NewStringCmd(context.Background()))
				mockRedis.EXPECT().ZAdd(gomock.Any(), gomock.Any(), gomock.Any()).Return(redis.NewIntCmd(context.Background()))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis = taskqueue.NewMockRedis(ctrl)

			tt.mock()

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
			if got == uuid.Nil {
				t.Errorf("TaskQueue.ProduceAt() = %v, want not nil", got)
			}
		})
	}
}
