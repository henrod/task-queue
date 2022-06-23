package taskqueue

import "time"

type Options struct {
	Namespace        string
	Address          string
	WorkerID         string
	MaxRetries       int
	OperationTimeout time.Duration
	QueueKey         string
}

func (o *Options) setDefaults() {
	if o.Namespace == "" {
		o.Namespace = "default"
	}

	if o.Address == "" {
		o.Address = "localhost:6379"
	}

	if o.WorkerID == "" {
		o.WorkerID = "worker0"
	}

	if o.MaxRetries == 0 {
		o.MaxRetries = -1
	}

	if o.OperationTimeout == 0 {
		o.OperationTimeout = 5 * time.Second
	}

	if o.QueueKey == "" {
		o.QueueKey = "taskqueue"
	}
}
