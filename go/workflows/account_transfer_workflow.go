package workflows

import (
	"fmt"
	"money-transfer-worker/activities"
	"money-transfer-worker/app"
	"money-transfer-worker/messages"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// LargeTransferThreshold is the amount above which approval is required
	LargeTransferThreshold = 1000
	// ApprovalTimeout is how long to wait for approval
	ApprovalTimeout = 60 * time.Second
)

func AccountTransferWorkflow(ctx workflow.Context, input app.TransferInput) (output *app.TransferOutput, err error) {
	name := workflow.GetInfo(ctx).WorkflowType.Name
	logger := workflow.GetLogger(ctx)
	logger.Info("Account Transfer workflow started", "name", name, "amount", input.Amount)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Expose transfer status as a query
	ts, err := messages.SetQueryHandlerForStatus(ctx)
	if err != nil {
		return nil, err
	}

	var idempotencyKey string
	_ = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return uuid.New().String()
	}).Get(&idempotencyKey)

	// Validate
	err = workflow.ExecuteActivity(ctx, activities.Validate, input).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, ts, 20)

	// Large transfer approval check
	if input.Amount > LargeTransferThreshold {
		logger.Info("Large transfer detected, requiring compliance approval",
			"amount", input.Amount,
			"threshold", LargeTransferThreshold,
			"workflowId", workflow.GetInfo(ctx).WorkflowExecution.ID)

		ts.TransferState = "waiting"
		ts.ProgressPercentage = 25
		ts.ApprovalTime = int(ApprovalTimeout.Seconds())

		approvalCh := messages.GetSignalChannelForApproval(ctx)
		approved, _ := approvalCh.ReceiveWithTimeout(ctx, ApprovalTimeout, nil)

		if !approved {
			ts.TransferState = "rejected"
			return nil, fmt.Errorf("transfer of $%d requires approval - not received within %v", input.Amount, ApprovalTimeout)
		}

		logger.Info("Large transfer approved", "amount", input.Amount)
	}
	updateProgress(ctx, 1, ts, 30)

	// Withdraw
	err = workflow.ExecuteActivity(ctx, activities.Withdraw, idempotencyKey, input.Amount, name).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	updateProgress(ctx, 2, ts, 55)

	// Deposit
	depositResponse := app.DepositResponse{}
	err = workflow.ExecuteActivity(ctx, activities.Deposit, idempotencyKey, input.Amount, name).Get(ctx, &depositResponse)
	if err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, ts, 80)

	// Send Notification
	err = workflow.ExecuteActivity(ctx, activities.SendNotification, input).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, ts, 100)

	output = &app.TransferOutput{
		DepositResponse: depositResponse,
	}
	return output, nil
}

func updateProgress(ctx workflow.Context, sleep int, ts *app.TransferStatus, progress int) {
	if sleep > 0 {
		duration := time.Duration(sleep) * time.Second
		workflow.Sleep(ctx, duration)
	}
	state := "running"
	if progress == 100 {
		state = "finished"
	}
	ts.TransferState = state
	ts.ProgressPercentage = progress
}
