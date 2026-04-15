package transfer

type SagaTransferInput struct {
	SenderAccountNumber   string  `json:"senderAccountNumber"`
	SenderRoutingNumber   string  `json:"senderRoutingNumber"`
	SenderName            string  `json:"senderName"`
	ReceiverAccountNumber string  `json:"receiverAccountNumber"`
	ReceiverRoutingNumber string  `json:"receiverRoutingNumber"`
	ReceiverName          string  `json:"receiverName"`
	Amount                float64 `json:"amount"`
	Reference             string  `json:"reference"`
}
