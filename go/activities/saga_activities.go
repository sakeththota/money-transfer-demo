package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"money-transfer-demo/balances"
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

type SagaActivities struct {
	DB *sql.DB
}

func (a *SagaActivities) ValidateAccounts(ctx context.Context, input transfer.SagaTransferInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating accounts",
		"sender", input.SenderAccountNumber,
		"receiver", input.ReceiverAccountNumber,
	)

	simulateExternalOperation(500)

	logger.Info("Account validation successful")
	return nil
}

func (a *SagaActivities) CheckBalance(ctx context.Context, fromAccount string, amount float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Checking balance", "account", fromAccount, "amount", amount)

	simulateExternalOperation(500)

	logger.Info("Sufficient balance confirmed")
	return nil
}

func (a *SagaActivities) DebitAccount(ctx context.Context, fromAccountName string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Debiting account", "account", fromAccountName, "amount", amount)

	simulateExternalOperation(1000)

	if err := balances.Debit(a.DB, fromAccountName, amount); err != nil {
		return "", err
	}

	txnRef := fmt.Sprintf("TXN-%s-%d", fromAccountName, activity.GetInfo(ctx).Attempt)
	logger.Info("Debit successful", "txnRef", txnRef)
	return txnRef, nil
}

func (a *SagaActivities) CreditAccount(ctx context.Context, toAccountName string, amount float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Crediting account", "account", toAccountName, "amount", amount)

	simulateExternalOperation(1000)

	logger.Error("Credit failed — recipient bank unavailable", "account", toAccountName)
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("credit failed: recipient bank unavailable for account %s", toAccountName),
		"CreditFailure",
		errors.New("bank service unavailable"),
	)
}

func (a *SagaActivities) RefundDebit(ctx context.Context, fromAccountName string, amount float64, originalTxnRef string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding debit (compensation)",
		"account", fromAccountName,
		"amount", amount,
		"originalTxnRef", originalTxnRef,
	)

	simulateExternalOperation(1000)

	if err := balances.Credit(a.DB, fromAccountName, amount); err != nil {
		return err
	}

	logger.Info("Refund successful", "originalTxnRef", originalTxnRef)
	return nil
}
