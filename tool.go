package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	wallettypes "decred.org/dcrwallet/rpc/jsonrpc/types"
	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
	"github.com/jrick/wsrpc/v2"
)

const (
	baseURL = "http://127.0.0.1:3000"

	rpcURL  = "wss://localhost:19110/ws"
	rpcUser = "test"
	rpcPass = "test"
)

var c *wsrpc.Client

func getPubKey() (*GetPubKeyResponse, error) {
	resp, err := http.Get(baseURL + "/api/pubkey")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Non 200 response from server: %v", string(b))
	}

	var j GetPubKeyResponse
	err = json.Unmarshal(b, &j)
	if err != nil {
		return nil, err
	}

	err = validateServerSignature(resp, b, j.PubKey)
	if err != nil {
		return nil, err
	}

	return &j, nil
}

func getFee(vspPubKey []byte) (*GetFeeResponse, error) {
	resp, err := http.Get(baseURL + "/api/fee")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Non 200 response from server: %v", string(b))
	}

	var j GetFeeResponse
	err = json.Unmarshal(b, &j)
	if err != nil {
		return nil, err
	}

	err = validateServerSignature(resp, b, vspPubKey)
	if err != nil {
		return nil, err
	}

	return &j, nil
}

func getFeeAddress(ticketHash string, commitmentAddr string, vspPubKey []byte) (*GetFeeAddressResponse, error) {
	req := GetFeeAddressRequest{
		TicketHash: ticketHash,
		Timestamp:  time.Now().Unix(),
	}
	resp, err := signedHTTP("/api/feeaddress", http.MethodPost, commitmentAddr, vspPubKey, req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("feeaddress response: %+v\n", string(resp))

	var j GetFeeAddressResponse
	err = json.Unmarshal(resp, &j)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func createFeeTx(feeAddress string, fee float64) (string, error) {
	amounts := make(map[string]float64)
	amounts[feeAddress] = 0.02

	var msgtxstr string
	err := c.Call(context.TODO(), "createrawtransaction", &msgtxstr, nil, amounts)
	if err != nil {
		return "", err
	}

	zero := int32(0)
	opt := wallettypes.FundRawTransactionOptions{
		ConfTarget: &zero,
	}
	var fundTx wallettypes.FundRawTransactionResult
	err = c.Call(context.TODO(), "fundrawtransaction", &fundTx, msgtxstr, "default", &opt)
	if err != nil {
		return "", err
	}

	var signedTx wallettypes.SignRawTransactionResult
	err = c.Call(context.TODO(), "signrawtransaction", &signedTx, fundTx.Hex)
	if err != nil {
		return "", err
	}
	if !signedTx.Complete {
		return "", fmt.Errorf("not all signed")
	}
	return signedTx.Hex, nil
}

func payFee(feeTx, privKeyWIF, ticketHash string, commitmentAddr string, vspPubKey []byte) error {
	req := PayFeeRequest{
		FeeTx:       feeTx,
		VotingKey:   privKeyWIF,
		TicketHash:  ticketHash,
		Timestamp:   time.Now().Unix(),
		VoteChoices: map[string]string{"headercommitments": "yes"},
	}

	_, err := signedHTTP("/api/payfee", http.MethodPost, commitmentAddr, vspPubKey, req)
	if err != nil {
		return err
	}

	return nil
}

func getTicketStatus(ticketHash string, commitmentAddr string, vspPubKey []byte) error {
	req := TicketStatusRequest{
		Timestamp:  time.Now().Unix(),
		TicketHash: ticketHash,
	}

	_, err := signedHTTP("/api/ticketstatus", http.MethodGet, commitmentAddr, vspPubKey, req)
	if err != nil {
		return err
	}

	return nil
}

func setVoteChoices(ticketHash string, commitmentAddr string, vspPubKey []byte, choices map[string]string) error {
	req := SetVoteChoicesRequest{
		Timestamp:   time.Now().Unix(),
		TicketHash:  ticketHash,
		VoteChoices: choices,
	}

	_, err := signedHTTP("/api/setvotechoices", http.MethodPost, commitmentAddr, vspPubKey, req)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	pubKeyResp, err := getPubKey()
	if err != nil {
		panic(err)
	}

	vspPubKey := pubKeyResp.PubKey

	fmt.Printf("pubkey: %x\n", vspPubKey)

	fee, err := getFee(vspPubKey)
	if err != nil {
		panic(err)
	}

	fmt.Printf("fee: %v\n", fee.Fee)

	ctx := context.Background()
	c, err = NewRPC(ctx, rpcURL, rpcUser, rpcPass)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// Get list of tickets
	var tickets wallettypes.GetTicketsResult
	includeImmature := true
	err = c.Call(ctx, "gettickets", &tickets, includeImmature)
	if err != nil {
		panic(err)
	}
	if len(tickets.Hashes) == 0 {
		panic("no tickets")
	}

	fmt.Printf("dcrwallet returned %d ticket(s):\n", len(tickets.Hashes))
	for _, tkt := range tickets.Hashes {
		fmt.Printf("    %s\n", tkt)
	}

	for i := 0; i < len(tickets.Hashes); i++ {
		privKeyStr, commitmentAddr, err := getPrivKeyAndCommitmentAddr(tickets.Hashes[i])
		if err != nil {
			panic(err)
		}

		feeAddress, err := getFeeAddress(tickets.Hashes[i], commitmentAddr, vspPubKey)
		if err != nil {
			panic(err)
		}
		if feeAddress == nil {
			continue
		}
		fmt.Printf("feeAddress: %v\n", feeAddress.FeeAddress)
		fmt.Printf("privKeyStr: %v\n", privKeyStr)

		feeTx, err := createFeeTx(feeAddress.FeeAddress, fee.Fee)
		if err != nil {
			fmt.Printf("createFeeTx error: %v\n", err)
			break
		}

		err = payFee(feeTx, privKeyStr, tickets.Hashes[i], commitmentAddr, vspPubKey)
		if err != nil {
			fmt.Printf("payFee error: %v\n", err)
			break
		}

		err = getTicketStatus(tickets.Hashes[i], commitmentAddr, vspPubKey)
		if err != nil {
			fmt.Printf("getTicketStatus error: %v\n", err)
			break
		}

		err = setVoteChoices(tickets.Hashes[i], commitmentAddr, vspPubKey, map[string]string{"headercommitments": "no"})
		if err != nil {
			fmt.Printf("setVoteChoices error: %v\n", err)
			break
		}

		err = getTicketStatus(tickets.Hashes[i], commitmentAddr, vspPubKey)
		if err != nil {
			fmt.Printf("getTicketStatus error: %v\n", err)
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func getPrivKeyAndCommitmentAddr(ticketHash string) (string, string, error) {
	var getTransactionResult wallettypes.GetTransactionResult
	err := c.Call(context.TODO(), "gettransaction", &getTransactionResult, ticketHash, false)
	if err != nil {
		fmt.Printf("gettransaction: %v\n", err)
		return "", "", err
	}

	msgTx := wire.NewMsgTx()
	if err = msgTx.Deserialize(hex.NewDecoder(strings.NewReader(getTransactionResult.Hex))); err != nil {
		return "", "", err
	}
	if len(msgTx.TxOut) < 2 {
		return "", "", errors.New("msgTx.TxOut < 2")
	}

	const scriptVersion = 0
	_, submissionAddr, _, err := txscript.ExtractPkScriptAddrs(scriptVersion,
		msgTx.TxOut[0].PkScript, chaincfg.TestNet3Params())
	if err != nil {
		return "", "", err
	}
	if len(submissionAddr) != 1 {
		return "", "", errors.New("submissionAddr != 1")
	}
	addr, err := stake.AddrFromSStxPkScrCommitment(msgTx.TxOut[1].PkScript,
		chaincfg.TestNet3Params())
	if err != nil {
		return "", "", err
	}

	var privKeyStr string
	err = c.Call(context.TODO(), "dumpprivkey", &privKeyStr, submissionAddr[0].Address())
	if err != nil {
		panic(err)
	}

	return privKeyStr, addr.Address(), nil
}

func signMessage(commitmentAddr string, msg []byte) (string, error) {
	var signature string
	err := c.Call(context.TODO(), "signmessage", &signature, commitmentAddr, string(msg))
	if err != nil {
		return "", err
	}
	return signature, nil
}
