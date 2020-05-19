package main

type GetPubKeyResponse struct {
	Timestamp int64  `json:"timestamp"`
	PubKey    []byte `json:"pubKey"`
}

type GetFeeResponse struct {
	Fee       float64 `json:"fee"`
	Signature []byte  `json:"signature"`
}

type GetFeeAddressRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"ticketHash"`
	Signature  string `json:"signature"`
}

type GetFeeAddressResponse struct {
	TicketHash          string `json:"ticketHash"`
	CommitmentSignature string `json:"commitmentSignature"`
	FeeAddress          string `json:"feeAddress"`
	Signature           []byte `json:"signature"`
}

type PayFeeRequest struct {
	Timestamp int64  `json:"timestamp"`
	Hex       string `json:"feeTx"`
	VotingKey string `json:"votingKey"`
	VoteBits  uint16 `json:"voteBits"`
}

type PayFeeResponse struct {
	Timestamp int64         `json:"timestamp"`
	TxHash    string        `json:"txHash"`
	Request   PayFeeRequest `json:"request"`
}
