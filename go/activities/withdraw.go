package activities

import (
	"context"
	"database/sql"
	"errors"
	"money-transfer-demo/balances"

	"go.temporal.io/sdk/activity"
)

type TransferActivities struct {
	DB *sql.DB
}

func (a *TransferActivities) Withdraw(ctx context.Context, idempotencyKey string, amount float64, scenario string, fromAccount string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Withdraw activity started", "amount", amount, "fromAccount", fromAccount)
	attempt := activity.GetInfo(ctx).Attempt

	if err := simulateExternalOperationWithError(1000, scenario, attempt); err != nil {
		logger.Info("Withdraw API unavailable", "attempt", attempt)
		return "", errors.New("withdraw activity failed, API unavailable")
	}

	if err := balances.Debit(a.DB, fromAccount, amount); err != nil {
		return "", err
	}

	logger.Info("Withdraw call complete", "fromAccount", fromAccount)
	return "SUCCESS", nil
}

func (a *TransferActivities) UndoWithdraw(ctx context.Context, amount float64, fromAccount string) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Undo Withdraw activity started", "amount", amount, "fromAccount", fromAccount)

	simulateExternalOperation(1000)

	if err := balances.Credit(a.DB, fromAccount, amount); err != nil {
		return false, err
	}

	return true, nil
}
