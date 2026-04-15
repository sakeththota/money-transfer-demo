package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/activity"
)

const (
	API_DOWNTIME = "AccountTransferWorkflowAPIDowntime"
)

func Withdraw(ctx context.Context, idempotencyKey string, amount float32, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Withdraw activity started", "amount", amount)
	attempt := activity.GetInfo(ctx).Attempt

	// simulate external API call
	error := simulateExternalOperationWithError(1000, name, attempt)
	logger.Info("Withdraw call complete", "name", name, "error", error)

	if API_DOWNTIME == error {
		// a transient error, which can be retried
		logger.Info("Withdraw API unavailable", "attempt", attempt)
		return "", errors.New("withdraw activity failed, API unavailable")
	}

	return "SUCCESS", nil
}

func UndoWithdraw(ctx context.Context, amount float32) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Undo Withdraw activity started", "amount", amount)

	// simulate external API call
	simulateExternalOperation(1000)

	return true, nil
}
