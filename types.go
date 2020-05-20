package main

type GetPubKeyResponse struct {
	Timestamp int64  `json:"timestamp"`
	PubKey    []byte `json:"pubkey"`
}

type GetFeeResponse struct {
	Fee       float64 `json:"fee"`
	Signature string  `json:"signature"`
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
	Signature           string `json:"signature"`
}

type PayFeeRequest struct {
	Timestamp   int64             `json:"timestamp"`
	TicketHash  string            `json:"tickethash"`
	FeeTx       string            `json:"feetx"`
	VotingKey   string            `json:"votingkey"`
	VoteChoices map[string]string `json:"votechoices"`
}

type PayFeeResponse struct {
	Timestamp int64         `json:"timestamp"`
	TxHash    string        `json:"txhash"`
	Request   PayFeeRequest `json:"request"`
}

type TicketStatusRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
	Signature  string `json:"signature"`
}

type SetVote struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
	Signature  string `json:"signature"`
}

type SetVoteChoicesRequest struct {
	Timestamp   int64             `json:"timestamp"`
	TicketHash  string            `json:"tickethash"`
	Signature   string            `json:"commitmentsignature"`
	VoteChoices map[string]string `json:"votechoices"`
}
