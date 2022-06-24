package taskqueue

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func ErrTaskLost(taskID uuid.UUID, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("task was permanently lost without being executed: %s, %w", taskID, err)
}

var (
	ErrMaxTaskReties   = errors.New("task reached max retries")
	ErrInvalidTaskType = errors.New("task not of type string")
	ErrNoTaskToConsume = errors.New("no task to consume")
)
