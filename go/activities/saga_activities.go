package activities

import (
	"context"
	"errors"
	"fmt"
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

type SagaActivities struct{}

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

func (a *SagaActivities) DebitAccount(ctx context.Context, fromAccount string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Debiting account", "account", fromAccount, "amount", amount)

	simulateExternalOperation(1000)

	txnRef := fmt.Sprintf("TXN-%s-%d", fromAccount, activity.GetInfo(ctx).Attempt)
	logger.Info("Debit successful", "txnRef", txnRef)
	return txnRef, nil
}

func (a *SagaActivities) CreditAccount(ctx context.Context, toAccount string, amount float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Crediting account", "account", toAccount, "amount", amount)

	simulateExternalOperation(1000)

	logger.Error("Credit failed — recipient bank unavailable", "account", toAccount)
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("credit failed: recipient bank unavailable for account %s", toAccount),
		"CreditFailure",
		errors.New("bank service unavailable"),
	)
}

func (a *SagaActivities) RefundDebit(ctx context.Context, fromAccount string, amount float64, originalTxnRef string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding debit (compensation)",
		"account", fromAccount,
		"amount", amount,
		"originalTxnRef", originalTxnRef,
	)

	simulateExternalOperation(1000)

	logger.Info("Refund successful", "originalTxnRef", originalTxnRef)
	return nil
}
