package transfer

type TransferInput struct {
	Amount               float64 `json:"amount"`
	FromAccount          string `json:"fromAccount"`
	FromAccountNumber    string `json:"fromAccountNumber"`
	FromRoutingNumber    string `json:"fromRoutingNumber"`
	ToAccount            string `json:"toAccount"`
	ToAccountNumber      string `json:"toAccountNumber"`
	ToRoutingNumber      string `json:"toRoutingNumber"`
}

type DepositResponse struct {
	DepositId string `json:"chargeId"`
}

type TransferOutput struct {
	DepositResponse DepositResponse `json:"depositResponse"`
}

type TransferStatus struct {
	ProgressPercentage int             `json:"progressPercentage"`
	TransferState      string          `json:"transferState"`
	WorkflowStatus     string          `json:"workflowStatus"`
	DepositResponse    DepositResponse `json:"chargeResult"`
	ApprovalTime       int             `json:"approvalTime"`
}
