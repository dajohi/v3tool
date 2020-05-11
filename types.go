package main

type GetPubKeyResponse struct {
	Timestamp int64  `json:"timestamp"`
	PubKey    []byte `json:"pubKey"`
}

type GetFeeResponse struct {
	Fee       float64 `json:"fee"`
	Signature []byte  `json:"signature"`
}

type GetFeeAddressResponse struct {
	TicketHash          string `json:"ticketHash"`
	CommitmentSignature string `json:"commitmentSignature"`
	FeeAddress          string `json:"feeAddress"`
	Signature           []byte `json:"signature"`
}
