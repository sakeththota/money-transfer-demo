package activities

import (
	"context"
	"money-transfer-worker/app"

	"go.temporal.io/sdk/activity"
)

func Deposit(ctx context.Context, idempotencyKey string, amount float32, name string) (app.DepositResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Deposit activity started", "amount", amount)
	attempt := activity.GetInfo(ctx).Attempt

	// simulate external API call
	simulateExternalOperationWithError(1000, name, attempt)
	logger.Info("Deposit call complete", "name", name)

	response := app.DepositResponse{
		DepositId: "example-transfer-id",
	}

	return response, nil
}
