package main

type GetVspInfoResponse struct {
	Timestamp     int64   `json:"timestamp"`
	PubKey        []byte  `json:"pubkey"`
	FeePercentage float64 `json:"feepercentage"`
	Closed        bool    `json:"closed"`
	Network       string  `json:"network"`
}

type GetFeeResponse struct {
	FeePercentage float64 `json:"feepercentage"`
}

type GetFeeAddressRequest struct {
	Timestamp  int64  `json:"timestamp"`
	TicketHash string `json:"tickethash"`
	TicketHex  string `json:"tickethex"`
}

type GetFeeAddressResponse struct {
	TicketHash string `json:"tickethash"`
	FeeAddress string `json:"feeaddress"`
	FeeAmount  int64  `json:"feeamount"`
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
	TicketHash string `json:"tickethash"`
}

type SetVoteChoicesRequest struct {
	Timestamp   int64             `json:"timestamp"`
	TicketHash  string            `json:"tickethash"`
	VoteChoices map[string]string `json:"votechoices"`
}
