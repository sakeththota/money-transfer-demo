package main

import (
	"context"
	"crypto/tls"
	"log"
	"log/slog"
	"money-transfer-demo/activities"
	"money-transfer-demo/encryption"
	"money-transfer-demo/workflows"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	tlog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	c, err := client.Dial(getClientOptions())
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, getEnv("TEMPORAL_MONEYTRANSFER_TASKQUEUE", "MoneyTransfer"), worker.Options{})

	// Register the same workflow function under different names for each scenario
	for _, name := range []string{
		workflows.HappyPath,
		workflows.AdvancedVisibility,
		workflows.HumanInLoop,
		workflows.APIDowntime,
		workflows.BugInWorkflow,
	} {
		w.RegisterWorkflowWithOptions(workflows.AccountTransferWorkflow, workflow.RegisterOptions{Name: name})
	}
	w.RegisterWorkflowWithOptions(workflows.SagaTransferWorkflow, workflow.RegisterOptions{
		Name: "AccountTransferWorkflowSagaRollback",
	})

	// activities
	w.RegisterActivity(activities.Validate)
	w.RegisterActivity(activities.Deposit)
	w.RegisterActivity(activities.Withdraw)
	w.RegisterActivity(activities.UndoWithdraw)
	w.RegisterActivity(activities.SendNotification)
	w.RegisterActivity(&activities.SagaActivities{})

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}

func getClientOptions() client.Options {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	address := getEnv("TEMPORAL_ADDRESS", "localhost:7233")
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")
	clientOptions := client.Options{
		HostPort:  address,
		Namespace: namespace,
		Logger:    tlog.NewStructuredLogger(logger),
	}

	apiKey := getEnv("TEMPORAL_API_KEY", "")
	tlsCertPath := getEnv("TEMPORAL_CERT_PATH", "")
	tlsKeyPath := getEnv("TEMPORAL_KEY_PATH", "")

	if apiKey != "" {
		clientOptions.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{},
			DialOptions: []grpc.DialOption{
				grpc.WithUnaryInterceptor(
					func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
						return invoker(
							metadata.AppendToOutgoingContext(ctx, "temporal-namespace", namespace),
							method,
							req,
							reply,
							cc,
							opts...,
						)
					},
				),
			},
		}

		clientOptions.Credentials = client.NewAPIKeyStaticCredentials(apiKey)

	} else if tlsCertPath != "" && tlsKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err != nil {
			log.Fatalln("Unable to load cert and key pair", err)
		}

		clientOptions.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
	}

	encryptPayloads := getEnv("ENCRYPT_PAYLOADS", "false")
	if encryptPayloads == "true" {
		encryptedDataConverter := encryption.NewEncryptionDataConverter(
			converter.GetDefaultDataConverter(),
			encryption.DataConverterOptions{KeyID: "test", Compress: false},
		)
		clientOptions.DataConverter = encryptedDataConverter
		clientOptions.FailureConverter = temporal.NewDefaultFailureConverter(temporal.DefaultFailureConverterOptions{
			DataConverter:          encryptedDataConverter,
			EncodeCommonAttributes: true,
		})
	}

	return clientOptions
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
