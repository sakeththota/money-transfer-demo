package activities

import (
	"context"
	"money-transfer-worker/app"

	"go.temporal.io/sdk/activity"
)

func SendNotification(ctx context.Context, input app.TransferInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Send notification activity started", "input", input)

	// simulate external API call
	simulateExternalOperation(1000)

	return "SUCCESS", nil
}
