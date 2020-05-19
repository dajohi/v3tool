package main

type GetPubKeyResponse struct {
	Timestamp int64  `json:"timestamp"`
	PubKey    []byte `json:"pubkey"`
}

type GetFeeResponse struct {
	Fee       float64 `json:"fee"`
	Signature []byte  `json:"signature"`
}

type GetFeeAddressRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
	Signature  string `json:"signature"`
}

type GetFeeAddressResponse struct {
	TicketHash          string `json:"tickethash"`
	CommitmentSignature string `json:"commitmentsignature"`
	FeeAddress          string `json:"feeaddress"`
	Signature           []byte `json:"signature"`
}

type PayFeeRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
	FeeTx      string `json:"feetx"`
	VotingKey  string `json:"votingkey"`
	VoteBits   uint16 `json:"votebits"`
}

type PayFeeResponse struct {
	Timestamp int64         `json:"timestamp"`
	TxHash    string        `json:"txhash"`
	Request   PayFeeRequest `json:"request"`
}
