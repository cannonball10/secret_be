package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// A list of task types.
const (
	TypeTestLog = "test:log"
)

type TestLogPayload struct {
	Message string
}

//----------------------------------------------
// Write a function NewXXXTask to create a task.
// A task consists of a type and a payload.
//----------------------------------------------

func NewTestLogTask(message string) (*asynq.Task, error) {
	payload, err := json.Marshal(TestLogPayload{Message: message})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeTestLog, payload), nil
}

//---------------------------------------------------------------
// Write a function HandleXXXTask to handle the input task.
// Note that it satisfies the asynq.HandlerFunc interface.
//
// Handler doesn't need to be a function. You can define a type
// that satisfies asynq.Handler interface. See examples below.
//---------------------------------------------------------------

func HandleTestLogTask(ctx context.Context, t *asynq.Task) error {
	var p TestLogPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	slog.InfoContext(ctx, "processing test log task", "message", p.Message)
	return nil
}
