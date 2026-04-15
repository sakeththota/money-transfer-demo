package messages

import "go.temporal.io/sdk/workflow"

// "approveTransfer" signal channel
func GetSignalChannelForApproval(ctx workflow.Context) workflow.ReceiveChannel {
	return workflow.GetSignalChannel(ctx, "approveTransfer")
}
