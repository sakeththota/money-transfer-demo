package workflows

import (
	"fmt"
	"money-transfer-demo/transfer"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func SagaTransferWorkflow(ctx workflow.Context, input transfer.SagaTransferInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Saga Transfer workflow started",
		"sender", input.SenderName,
		"receiver", input.ReceiverName,
		"amount", input.Amount,
	)

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
		},
	})

	// Unified query handler — same name and shape as all other scenarios
	ts := &transfer.TransferStatus{TransferState: "running"}
	err := workflow.SetQueryHandler(ctx, "transferStatus", func() (transfer.TransferStatus, error) {
		return *ts, nil
	})
	if err != nil {
		return err
	}

	var compensations []func(ctx workflow.Context) error

	// Compensations run in reverse order using NewDisconnectedContext
	// so they execute even if the workflow is cancelled
	runCompensations := func() {
		disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
		compCtx := workflow.WithActivityOptions(disconnectedCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
		})
		for i := len(compensations) - 1; i >= 0; i-- {
			if err := compensations[i](compCtx); err != nil {
				logger.Error("Compensation failed", "error", err)
			}
		}
	}

	// Step 1: ValidateAccounts (no compensation — read-only)
	workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("ValidateAccounts"))
	err = workflow.ExecuteActivity(actCtx, "ValidateAccounts", input).Get(ctx, nil)
	if err != nil {
		return err
	}
	updateProgress(ctx, 1, ts, 20)

	// Step 2: CheckBalance (no compensation — read-only)
	workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("CheckBalance"))
	err = workflow.ExecuteActivity(actCtx, "CheckBalance", input.SenderAccountNumber, input.Amount).Get(ctx, nil)
	if err != nil {
		return err
	}
	updateProgress(ctx, 1, ts, 40)

	// Step 3: DebitAccount — register compensation BEFORE executing
	// If the activity completes the effect but fails on return, we still need compensation
	workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("DebitAccount"))
	var debitTxnRef string
	compensations = append(compensations, func(ctx workflow.Context) error {
		return workflow.ExecuteActivity(ctx, "RefundDebit", input.SenderAccountNumber, input.Amount, debitTxnRef).Get(ctx, nil)
	})
	err = workflow.ExecuteActivity(actCtx, "DebitAccount", input.SenderAccountNumber, input.Amount).Get(ctx, &debitTxnRef)
	if err != nil {
		runCompensations()
		return err
	}
	updateProgress(ctx, 1, ts, 60)

	// Step 4: CreditAccount — FAILS by design in this scenario
	workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("CreditAccount"))
	err = workflow.ExecuteActivity(actCtx, "CreditAccount", input.ReceiverAccountNumber, input.Amount).Get(ctx, nil)
	if err != nil {
		logger.Info("CreditAccount failed, initiating saga compensation", "error", err)

		workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("Compensating"))
		ts.TransferState = "compensating"
		ts.ProgressPercentage = 80
		runCompensations()

		workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("Compensated"))
		ts.TransferState = "compensated"
		ts.ProgressPercentage = 100

		return fmt.Errorf("transfer rolled back: %w", err)
	}
	updateProgress(ctx, 1, ts, 80)

	workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet("Completed"))
	updateProgress(ctx, 0, ts, 100)
	return nil
}
