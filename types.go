package main

type GetPubKeyResponse struct {
	Timestamp int64  `json:"timestamp"`
	PubKey    []byte `json:"pubkey"`
}

type GetFeeResponse struct {
	FeePercentage float64 `json:"feepercentage"`
}

type GetFeeAddressRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
}

type GetFeeAddressResponse struct {
	TicketHash string  `json:"tickethash"`
	FeeAddress string  `json:"feeaddress"`
	FeeAmount  float64 `json:"feeamount"`
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
}

type SetVoteChoicesRequest struct {
	Timestamp   int64             `json:"timestamp"`
	TicketHash  string            `json:"tickethash"`
	VoteChoices map[string]string `json:"votechoices"`
}
