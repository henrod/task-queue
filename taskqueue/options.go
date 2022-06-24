package taskqueue

import "time"

type Options struct {
	QueueKey         string
	Namespace        string
	StorageAddress   string
	WorkerID         string
	MaxRetries       int
	OperationTimeout time.Duration
}

func (o *Options) setDefaults() {
	if o.Namespace == "" {
		o.Namespace = "default"
	}

	if o.StorageAddress == "" {
		o.StorageAddress = "localhost:6379"
	}

	if o.WorkerID == "" {
		o.WorkerID = "worker0"
	}

	if o.MaxRetries == 0 {
		o.MaxRetries = -1
	}

	if o.OperationTimeout == 0 {
		o.OperationTimeout = 5 * time.Second // nolint:gomnd
	}

	if o.QueueKey == "" {
		o.QueueKey = "taskqueue"
	}
}
