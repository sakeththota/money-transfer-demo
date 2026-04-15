package activities

import (
	"errors"
	"time"
)

const (
	APIDowntime = "AccountTransferWorkflowAPIDowntime"
)

var ErrAPIUnavailable = errors.New("API unavailable")

func simulateExternalOperation(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func simulateExternalOperationWithError(ms int, name string, attempt int32) error {
	simulateExternalOperation(ms / int(attempt))
	// Simulate API downtime for first 4 attempts when running APIDowntime scenario
	if name == APIDowntime && attempt < 5 {
		return ErrAPIUnavailable
	}
	return nil
}
