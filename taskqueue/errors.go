package taskqueue

import (
	"fmt"

	"github.com/google/uuid"
)

func ErrTaskLost(taskID uuid.UUID, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("task was permanentely lost without being executed: %s, %w", taskID, err)
}
