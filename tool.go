package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
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

func getPubKey() *GetPubKeyResponse {
	resp, err := http.Get(baseURL + "/api/pubkey")
	if err != nil {
		panic(err)
	}
	sigStr := resp.Header.Get("VSP-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}

	var j GetPubKeyResponse
	err = json.Unmarshal(b, &j)
	if err != nil {
		panic(err)
	}

	if !ed25519.Verify(j.PubKey, b, sig) {
		panic("bad signature")
	}

	return &j
}

func getFee(pubKey ed25519.PublicKey) *GetFeeResponse {
	resp, err := http.Get(baseURL + "/api/fee")
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}
	sigStr := resp.Header.Get("VSP-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		panic(err)
	}

	if !ed25519.Verify(pubKey, b, sig) {
		panic("bad signature")
	}

	var j GetFeeResponse
	err = json.Unmarshal(b, &j)
	if err != nil {
		panic(err)
	}

	return &j
}

func getFeeAddress(ctx context.Context, c *wsrpc.Client, pubKey ed25519.PublicKey, ticket string) (*GetFeeAddressResponse, string) {
	msg := fmt.Sprintf("vsp v3 getfeeaddress %s", ticket)
	signature, privKeyStr, err := signMsgGetPrivKey(ctx, c, ticket, msg)
	if err != nil {
		panic(err)
	}

	reqBytes, err := json.Marshal(GetFeeAddressRequest{
		TicketHash: ticket,
		Signature:  signature,
		Timestamp:  time.Now().Unix(),
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/feeaddress", bytes.NewBuffer(reqBytes))
	if err != nil {
		panic(err)
	}

	var httpClient http.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}
	fmt.Printf("feeaddress response: %+v\n", string(b))

	sigStr := resp.Header.Get("VSP-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		panic(err)
	}

	if !ed25519.Verify(pubKey, b, sig) {
		panic("bad signature")
	}

	var j GetFeeAddressResponse
	err = json.Unmarshal(b, &j)
	if err != nil {
		panic(err)
	}
	return &j, privKeyStr
}

func payFee(ctx context.Context, c *wsrpc.Client, privKeyWIF string, pubKey ed25519.PublicKey, ticketHash string, address string, fee float64) error {
	fmt.Printf("payfee...\n")

	amounts := make(map[string]float64)
	amounts[address] = 0.02

	var msgtxstr string
	err := c.Call(ctx, "createrawtransaction", &msgtxstr, nil, amounts)
	if err != nil {
		return err
	}

	zero := int32(0)
	opt := wallettypes.FundRawTransactionOptions{
		ConfTarget: &zero,
	}
	var fundTx wallettypes.FundRawTransactionResult
	err = c.Call(ctx, "fundrawtransaction", &fundTx, msgtxstr, "default", &opt)
	if err != nil {
		return err
	}

	var signedTx wallettypes.SignRawTransactionResult
	err = c.Call(ctx, "signrawtransaction", &signedTx, fundTx.Hex)
	if err != nil {
		return err
	}
	if !signedTx.Complete {
		return fmt.Errorf("not all signed")
	}

	reqBytes, err := json.Marshal(PayFeeRequest{
		FeeTx:       signedTx.Hex,
		VotingKey:   privKeyWIF,
		TicketHash:  ticketHash,
		Timestamp:   time.Now().Unix(),
		VoteChoices: map[string]string{"headercommitments": "yes"},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/payfee", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}

	var httpClient http.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	fmt.Printf("payfee response: %+v\n", string(b))

	sigStr := resp.Header.Get("VSP-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, b, sig) {
		panic("bad signature")
	}

	// var hash string
	// err = c.Call(ctx, "sendrawtransaction", &hash, signedTx.Hex)
	// if err != nil {
	// 	fmt.Printf("failed to send tx: %v\n", err)
	// }
	// fmt.Printf("%s\n", hash)
	// fmt.Printf("%s\n", string(b))
	return nil
}

func getTicketStatus(ctx context.Context, c *wsrpc.Client, ticketHash string) error {
	timestamp := time.Now().Unix()
	msg := fmt.Sprintf("vsp v3 ticketstatus %d %s", timestamp, ticketHash)
	signature, _, err := signMsgGetPrivKey(ctx, c, ticketHash, msg)
	if err != nil {
		return err
	}

	reqBytes, err := json.Marshal(TicketStatusRequest{
		Timestamp:  timestamp,
		TicketHash: ticketHash,
		Signature:  signature,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/ticketstatus", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}

	var httpClient http.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	fmt.Printf("ticketstatus response: %+v\n", string(b))
	return nil
}

func main() {
	pubKey := getPubKey()
	fmt.Printf("pubkey: %x\n", pubKey.PubKey)

	fee := getFee(pubKey.PubKey)
	fmt.Printf("fee: %v\n", fee.Fee)

	ctx := context.Background()
	c, err := NewRPC(ctx, rpcURL, rpcUser, rpcPass)
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
		feeAddress, privKeyStr := getFeeAddress(ctx, c, pubKey.PubKey, tickets.Hashes[i])
		if feeAddress == nil {
			continue
		}
		fmt.Printf("feeAddress: %v\n", feeAddress.FeeAddress)
		fmt.Printf("privKeyStr: %v\n", privKeyStr)

		err := payFee(ctx, c, privKeyStr, pubKey.PubKey, tickets.Hashes[i], feeAddress.FeeAddress, fee.Fee)
		if err != nil {
			fmt.Printf("payFee error: %v\n", err)
			break
		}

		err = getTicketStatus(ctx, c, tickets.Hashes[i])
		if err != nil {
			fmt.Printf("getTicketStatus error: %v\n", err)
			break
		}

		err = setVoteChoices(ctx, c, tickets.Hashes[i], map[string]string{"headercommitments": "no"})
		if err != nil {
			fmt.Printf("setVoteChoices error: %v\n", err)
			break
		}

		err = getTicketStatus(ctx, c, tickets.Hashes[i])
		if err != nil {
			fmt.Printf("getTicketStatus error: %v\n", err)
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func setVoteChoices(ctx context.Context, c *wsrpc.Client, ticketHash string, choices map[string]string) error {
	timestamp := time.Now().Unix()
	msg := fmt.Sprintf("vsp v3 setvotechoices %d %s %v", timestamp, ticketHash, choices)
	signature, _, err := signMsgGetPrivKey(ctx, c, ticketHash, msg)
	if err != nil {
		return err
	}

	reqBytes, err := json.Marshal(SetVoteChoicesRequest{
		Timestamp:   timestamp,
		TicketHash:  ticketHash,
		Signature:   signature,
		VoteChoices: choices,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/setvotechoices", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}

	var httpClient http.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	fmt.Printf("setvotechoices response: %+v\n", string(b))
	return nil
}

func signMsgGetPrivKey(ctx context.Context, c *wsrpc.Client, ticketHash, msg string) (string, string, error) {
	var getTransactionResult wallettypes.GetTransactionResult
	err := c.Call(ctx, "gettransaction", &getTransactionResult, ticketHash, false)
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

	var signature string
	err = c.Call(ctx, "signmessage", &signature, addr.Address(), msg)
	if err != nil {
		return "", "", err
	}

	var privKeyStr string
	err = c.Call(ctx, "dumpprivkey", &privKeyStr, submissionAddr[0].Address())
	if err != nil {
		panic(err)
	}

	return signature, privKeyStr, nil
}
