package taskqueue

import "github.com/sirupsen/logrus"

const (
	labelPackage    = "package"
	labelTaskID     = "task_id"
	labelRetryCount = "retry_count"
)

func newLogger() logrus.FieldLogger {
	return logrus.New().WithFields(logrus.Fields{
		labelPackage: "taskqueue",
	})
}

func withTaskLabels(logger logrus.FieldLogger, task *Task) logrus.FieldLogger {
	return logger.WithFields(logrus.Fields{
		labelTaskID:     task.ID,
		labelRetryCount: task.RetryCount,
	})
}
