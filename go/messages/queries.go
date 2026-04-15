package messages

import (
	"money-transfer-worker/app"

	"go.temporal.io/sdk/workflow"
)

// "transferStatus" query handler
func SetQueryHandlerForStatus(ctx workflow.Context) (*app.TransferStatus, error) {
	logger := workflow.GetLogger(ctx)

	// Initialize our state object
	var state = app.TransferStatus{}
	state.ProgressPercentage = 0
	state.TransferState = "starting"
	state.WorkflowStatus = ""
	state.ApprovalTime = 30
	state.DepositResponse = app.DepositResponse{}

	err := workflow.SetQueryHandler(ctx, "transferStatus", func() (app.TransferStatus, error) {
		return state, nil
	})
	if err != nil {
		logger.Error("SetQueryHandler failed for transferStatus: " + err.Error())
		return nil, err
	}

	return &state, nil
}
