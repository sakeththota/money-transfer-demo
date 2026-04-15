package activities

import (
	"context"
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/activity"
)

func Deposit(ctx context.Context, idempotencyKey string, amount float32, name string) (transfer.DepositResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Deposit activity started", "amount", amount)
	attempt := activity.GetInfo(ctx).Attempt

	// simulate external API call
	if err := simulateExternalOperationWithError(1000, name, attempt); err != nil {
		return transfer.DepositResponse{}, err
	}
	logger.Info("Deposit call complete", "name", name)

	response := transfer.DepositResponse{
		DepositId: "example-transfer-id",
	}

	return response, nil
}
