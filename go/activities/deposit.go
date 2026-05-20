package activities

import (
	"context"
	"money-transfer-demo/balances"
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/activity"
)

func (a *TransferActivities) Deposit(ctx context.Context, idempotencyKey string, amount float64, scenario string, toAccount string) (transfer.DepositResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Deposit activity started", "amount", amount, "toAccount", toAccount)
	attempt := activity.GetInfo(ctx).Attempt

	if err := simulateExternalOperationWithError(1000, scenario, attempt); err != nil {
		return transfer.DepositResponse{}, err
	}

	if err := balances.Credit(a.DB, toAccount, amount); err != nil {
		return transfer.DepositResponse{}, err
	}

	logger.Info("Deposit call complete", "toAccount", toAccount)
	return transfer.DepositResponse{DepositId: "example-transfer-id"}, nil
}
