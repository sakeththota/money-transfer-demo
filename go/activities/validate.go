package activities

import (
	"context"
	"money-transfer-demo/transfer"

	"go.temporal.io/sdk/activity"
)

func Validate(ctx context.Context, input transfer.TransferInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validate activity started", "input", input)

	// simulate external API call
	simulateExternalOperation(1000)

	return "SUCCESS", nil
}
