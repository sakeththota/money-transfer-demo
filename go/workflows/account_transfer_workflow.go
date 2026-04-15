package workflows

import (
	"fmt"
	"money-transfer-demo/activities"
	"money-transfer-demo/transfer"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Workflow type names (registered in worker)
const (
	HappyPath          = "AccountTransferWorkflow"
	AdvancedVisibility = "AccountTransferWorkflowAdvancedVisibility"
	HumanInLoop        = "AccountTransferWorkflowHumanInLoop"
	APIDowntime        = "AccountTransferWorkflowAPIDowntime"
	BugInWorkflow      = "AccountTransferWorkflowRecoverableFailure"
)

const (
	LargeTransferThreshold = 1000
	ApprovalTimeout        = 30 * time.Second
)

var stepKey = temporal.NewSearchAttributeKeyKeyword("Step")

// AccountTransferWorkflow handles all money transfer scenarios.
// Behavior varies based on the registered workflow type name.
func AccountTransferWorkflow(ctx workflow.Context, input transfer.TransferInput) (*transfer.TransferOutput, error) {
	scenario := workflow.GetInfo(ctx).WorkflowType.Name
	logger := workflow.GetLogger(ctx)
	logger.Info("Account Transfer workflow started", "scenario", scenario, "amount", input.Amount)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
		},
	})

	// Set up query handler for status tracking
	status, err := SetupStatusQuery(ctx)
	if err != nil {
		return nil, err
	}

	var idempotencyKey string
	_ = workflow.SideEffect(ctx, func(ctx workflow.Context) any {
		return uuid.New().String()
	}).Get(&idempotencyKey)

	// Step 1: Validate
	upsertStep(ctx, scenario, "Validate")
	if err := workflow.ExecuteActivity(ctx, activities.Validate, input).Get(ctx, nil); err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, status, 20)

	// Step 2: Approval (scenario-dependent)
	if err := handleApproval(ctx, scenario, input.Amount, status); err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, status, 30)

	// Step 3: Withdraw
	upsertStep(ctx, scenario, "Withdraw")
	if err := workflow.ExecuteActivity(ctx, activities.Withdraw, idempotencyKey, input.Amount, scenario).Get(ctx, nil); err != nil {
		return nil, err
	}
	updateProgress(ctx, 2, status, 50)

	// Bug simulation (only for BugInWorkflow scenario)
	if scenario == BugInWorkflow {
		panic("Simulated bug - fix me!")
	}

	// Step 4: Deposit
	upsertStep(ctx, scenario, "Deposit")
	var depositResponse transfer.DepositResponse
	if err := workflow.ExecuteActivity(ctx, activities.Deposit, idempotencyKey, input.Amount, scenario).Get(ctx, &depositResponse); err != nil {
		logger.Info("Deposit failed, reverting withdraw")
		_ = workflow.ExecuteActivity(ctx, activities.UndoWithdraw, input.Amount).Get(ctx, nil)
		return nil, fmt.Errorf("deposit failed: %w", err)
	}
	updateProgress(ctx, 1, status, 75)

	// Step 5: Send Notification
	upsertStep(ctx, scenario, "SendNotification")
	if err := workflow.ExecuteActivity(ctx, activities.SendNotification, input).Get(ctx, nil); err != nil {
		return nil, err
	}
	updateProgress(ctx, 1, status, 100)

	return &transfer.TransferOutput{DepositResponse: depositResponse}, nil
}

// handleApproval manages approval logic based on scenario
func handleApproval(ctx workflow.Context, scenario string, amount int, status *transfer.TransferStatus) error {
	logger := workflow.GetLogger(ctx)
	needsApproval := false
	timeout := ApprovalTimeout

	switch scenario {
	case HumanInLoop:
		// Always requires approval
		needsApproval = true
		logger.Info("Human-in-loop scenario: waiting for approval signal",
			"workflowId", workflow.GetInfo(ctx).WorkflowExecution.ID)
	default:
		// Large transfers require approval
		if amount > LargeTransferThreshold {
			needsApproval = true
			logger.Info("Large transfer requires approval",
				"amount", amount,
				"threshold", LargeTransferThreshold)
		}
	}

	if !needsApproval {
		return nil
	}

	status.TransferState = "waiting"
	status.ApprovalTime = int(timeout.Seconds())

	approved, _ := GetApprovalChannel(ctx).ReceiveWithTimeout(ctx, timeout, nil)
	if !approved {
		status.TransferState = "rejected"
		return fmt.Errorf("approval not received within %v", timeout)
	}

	logger.Info("Transfer approved")
	return nil
}

// upsertStep updates search attributes for visibility demo
func upsertStep(ctx workflow.Context, scenario, step string) {
	if scenario == AdvancedVisibility {
		workflow.GetLogger(ctx).Info("Updating search attribute", "step", step)
		workflow.UpsertTypedSearchAttributes(ctx, stepKey.ValueSet(step))
	}
}

// updateProgress updates the query-able status
func updateProgress(ctx workflow.Context, sleepSec int, status *transfer.TransferStatus, progress int) {
	if sleepSec > 0 {
		workflow.Sleep(ctx, time.Duration(sleepSec)*time.Second)
	}
	status.ProgressPercentage = progress
	if progress == 100 {
		status.TransferState = "finished"
	} else if status.TransferState != "waiting" {
		status.TransferState = "running"
	}
}
