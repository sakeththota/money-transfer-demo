package workflows

import (
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/workflow"
)

// GetApprovalChannel returns the signal channel for transfer approval
func GetApprovalChannel(ctx workflow.Context) workflow.ReceiveChannel {
	return workflow.GetSignalChannel(ctx, "approveTransfer")
}

// SetupStatusQuery sets up the "transferStatus" query handler and returns a pointer
// to the status struct that workflows can update as they progress
func SetupStatusQuery(ctx workflow.Context) (*transfer.TransferStatus, error) {
	logger := workflow.GetLogger(ctx)

	state := transfer.TransferStatus{
		ProgressPercentage: 0,
		TransferState:      "starting",
		WorkflowStatus:     "",
		ApprovalTime:       30,
		DepositResponse:    transfer.DepositResponse{},
	}

	err := workflow.SetQueryHandler(ctx, "transferStatus", func() (transfer.TransferStatus, error) {
		return state, nil
	})
	if err != nil {
		logger.Error("SetQueryHandler failed for transferStatus: " + err.Error())
		return nil, err
	}

	return &state, nil
}
