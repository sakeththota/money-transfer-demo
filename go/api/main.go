package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"money-transfer-worker/app"
	"money-transfer-worker/encryption"
)

var temporalClient client.Client

func main() {
	var err error
	temporalClient, err = client.Dial(getClientOptions())
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer temporalClient.Close()

	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:7070"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	r.GET("/serverinfo", handleServerInfo)
	r.POST("/runWorkflow", handleRunWorkflow)
	r.POST("/runQuery", handleRunQuery)
	r.POST("/approveTransfer", handleApproveTransfer)
	r.GET("/listWorkflows", handleListWorkflows)
	r.POST("/scheduleWorkflow", handleScheduleWorkflow)
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello from Go API!")
	})

	port := getEnv("API_PORT", "7070")
	log.Printf("Starting API server on port %s", port)
	r.Run(":" + port)
}

// ServerInfo response
type ServerInfo struct {
	Address          string `json:"address"`
	Namespace        string `json:"namespace"`
	TaskQueue        string `json:"taskQueue"`
	EncryptPayloads  bool   `json:"encryptPayloads"`
	SecureConnection bool   `json:"secureConnection"`
	CodecServerURL   string `json:"codecServerUrl,omitempty"`
}

func handleServerInfo(c *gin.Context) {
	encryptPayloads := getEnv("ENCRYPT_PAYLOADS", "false") == "true"
	// Secure connection if using mTLS or API key (both use TLS)
	hasMTLS := getEnv("TEMPORAL_CERT_PATH", "") != "" && getEnv("TEMPORAL_KEY_PATH", "") != ""
	hasAPIKey := getEnv("TEMPORAL_API_KEY", "") != ""
	secureConnection := hasMTLS || hasAPIKey

	info := ServerInfo{
		Address:          getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace:        getEnv("TEMPORAL_NAMESPACE", "default"),
		TaskQueue:        getEnv("TEMPORAL_MONEYTRANSFER_TASKQUEUE", "MoneyTransfer"),
		EncryptPayloads:  encryptPayloads,
		SecureConnection: secureConnection,
	}
	if encryptPayloads {
		info.CodecServerURL = getEnv("CODEC_SERVER_URL", "http://localhost:8081")
	}
	c.JSON(http.StatusOK, info)
}

// UXParameters from frontend
type UXParameters struct {
	Amount      int    `json:"amount"`
	FromAccount string `json:"fromAccount"`
	ToAccount   string `json:"toAccount"`
	Scenario    string `json:"scenario"`
}

// Scenario to workflow type mapping
var scenarioToWorkflow = map[string]string{
	"HAPPY_PATH":          "AccountTransferWorkflow",
	"ADVANCED_VISIBILITY": "AccountTransferWorkflowAdvancedVisibility",
	"HUMAN_IN_LOOP":       "AccountTransferWorkflowHumanInLoop",
	"API_DOWNTIME":        "AccountTransferWorkflowAPIDowntime",
	"BUG_IN_WORKFLOW":     "AccountTransferWorkflowRecoverableFailure",
	"SAGA_ROLLBACK":       "AccountTransferWorkflowSagaRollback",
}

func handleRunWorkflow(c *gin.Context) {
	var params UXParameters
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflowType, ok := scenarioToWorkflow[params.Scenario]
	if !ok {
		workflowType = "AccountTransferWorkflow"
	}

	referenceNumber := generateReferenceNumber()
	taskQueue := getEnv("TEMPORAL_MONEYTRANSFER_TASKQUEUE", "MoneyTransfer")

	options := client.StartWorkflowOptions{
		ID:        referenceNumber,
		TaskQueue: taskQueue,
	}

	var we client.WorkflowRun
	var err error

	if params.Scenario == "SAGA_ROLLBACK" {
		fromAcctNum, fromRoutingNum := getAccountDetails(params.FromAccount)
		toAcctNum, toRoutingNum := getAccountDetails(params.ToAccount)
		sagaInput := app.SagaTransferInput{
			SenderAccountNumber:   fromAcctNum,
			SenderRoutingNumber:   fromRoutingNum,
			SenderName:            params.FromAccount,
			ReceiverAccountNumber: toAcctNum,
			ReceiverRoutingNumber: toRoutingNum,
			ReceiverName:          params.ToAccount,
			Amount:                float64(params.Amount),
			Reference:             referenceNumber,
		}
		we, err = temporalClient.ExecuteWorkflow(context.Background(), options, workflowType, sagaInput)
	} else {
		fromAcctNum, fromRoutingNum := getAccountDetails(params.FromAccount)
		toAcctNum, toRoutingNum := getAccountDetails(params.ToAccount)
		transferInput := app.TransferInput{
			Amount:            params.Amount,
			FromAccount:       params.FromAccount,
			FromAccountNumber: fromAcctNum,
			FromRoutingNumber: fromRoutingNum,
			ToAccount:         params.ToAccount,
			ToAccountNumber:   toAcctNum,
			ToRoutingNumber:   toRoutingNum,
		}
		we, err = temporalClient.ExecuteWorkflow(context.Background(), options, workflowType, transferInput)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Started workflow %s with ID %s", workflowType, we.GetID())
	c.JSON(http.StatusOK, gin.H{"transferId": referenceNumber})
}

type WorkflowIdRequest struct {
	WorkflowId string `json:"workflowId"`
}

func handleRunQuery(c *gin.Context) {
	var req WorkflowIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get workflow status
	workflowStatus := getWorkflowExecutionStatus(req.WorkflowId)

	// All workflows use the same "transferStatus" query
	resp, err := temporalClient.QueryWorkflow(context.Background(), req.WorkflowId, "", "transferStatus")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"progressPercentage": 0,
			"transferState":      "unknown",
			"workflowStatus":     workflowStatus,
		})
		return
	}

	var status app.TransferStatus
	if err := resp.Get(&status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Always set the workflow execution status
	status.WorkflowStatus = workflowStatus

	c.JSON(http.StatusOK, status)
}

func handleApproveTransfer(c *gin.Context) {
	var req WorkflowIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := temporalClient.SignalWorkflow(context.Background(), req.WorkflowId, "", "approveTransfer", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signal": "sent"})
}

type WorkflowStatus struct {
	WorkflowId string `json:"workflowId"`
	Status     string `json:"status"`
}

func handleListWorkflows(c *gin.Context) {
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")

	// List all workflows for this task queue
	query := fmt.Sprintf("TaskQueue = '%s'",
		getEnv("TEMPORAL_MONEYTRANSFER_TASKQUEUE", "MoneyTransfer"))

	resp, err := temporalClient.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: namespace,
		Query:     query,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var workflows []WorkflowStatus
	for _, execution := range resp.Executions {
		workflows = append(workflows, WorkflowStatus{
			WorkflowId: execution.Execution.WorkflowId,
			Status:     execution.Status.String(),
		})
	}

	c.JSON(http.StatusOK, workflows)
}

type ScheduleParameters struct {
	Amount        int    `json:"amount"`
	FromAccount   string `json:"fromAccount"`
	ToAccount     string `json:"toAccount"`
	Scenario      string `json:"scenario"`
	IntervalHours int    `json:"intervalHours"`
}

func handleScheduleWorkflow(c *gin.Context) {
	var params ScheduleParameters
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflowType, ok := scenarioToWorkflow[params.Scenario]
	if !ok {
		workflowType = "AccountTransferWorkflow"
	}

	scheduleID := fmt.Sprintf("schedule-%s", generateReferenceNumber())
	taskQueue := getEnv("TEMPORAL_MONEYTRANSFER_TASKQUEUE", "MoneyTransfer")

	fromAcctNum, fromRoutingNum := getAccountDetails(params.FromAccount)
	toAcctNum, toRoutingNum := getAccountDetails(params.ToAccount)
	transferInput := app.TransferInput{
		Amount:            params.Amount,
		FromAccount:       params.FromAccount,
		FromAccountNumber: fromAcctNum,
		FromRoutingNumber: fromRoutingNum,
		ToAccount:         params.ToAccount,
		ToAccountNumber:   toAcctNum,
		ToRoutingNumber:   toRoutingNum,
	}

	interval := time.Duration(params.IntervalHours) * time.Hour
	if params.IntervalHours == 0 {
		interval = 24 * time.Hour
	}

	_, err := temporalClient.ScheduleClient().Create(context.Background(), client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{Every: interval},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-{{.ScheduledTime.Format \"20060102T150405\"}}", scheduleID),
			Workflow:  workflowType,
			Args:      []interface{}{transferInput},
			TaskQueue: taskQueue,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transferId": scheduleID})
}

func generateReferenceNumber() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return fmt.Sprintf("TRANSFER-%c%c%c-%03d",
		chars[rand.Intn(26)],
		chars[rand.Intn(26)],
		chars[rand.Intn(26)],
		rand.Intn(999))
}

// Simulated account data - in a real app this would come from a database
var accountData = map[string]struct {
	AccountNumber string
	RoutingNumber string
}{
	"Checking Account": {"4532-8901-2345-6789", "021000021"},
	"Savings Account":  {"4532-1122-3344-5566", "021000021"},
	"Justine Morris":   {"9876-5432-1098-7654", "026009593"},
	"Raul Ruidiaz":     {"5544-3322-1100-9988", "021000089"},
	"Ian Wu":           {"1234-5678-9012-3456", "071000013"},
	"Emma Stockton":    {"6677-8899-0011-2233", "121000248"},
}

func getAccountDetails(name string) (accountNumber, routingNumber string) {
	if data, ok := accountData[name]; ok {
		return data.AccountNumber, data.RoutingNumber
	}
	// Default fallback
	return "0000-0000-0000-0000", "000000000"
}

func getWorkflowExecutionStatus(workflowId string) string {
	resp, err := temporalClient.DescribeWorkflowExecution(context.Background(), workflowId, "")
	if err != nil {
		return "UNKNOWN"
	}

	switch resp.WorkflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "RUNNING"
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "COMPLETED"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "FAILED"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "CANCELED"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return "TERMINATED"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "TIMED_OUT"
	default:
		return "UNKNOWN"
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

	// Enable encryption if requested
	encryptPayloads := getEnv("ENCRYPT_PAYLOADS", "false")
	if encryptPayloads == "true" {
		log.Println("Payload encryption enabled")
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
