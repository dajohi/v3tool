package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
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
	baseURL = "https://teststakepool.decred.org"

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

	var privKeyStr string
	var getTransactionResult wallettypes.GetTransactionResult
	err := c.Call(ctx, "gettransaction", &getTransactionResult, ticket, false)
	if err != nil {
		fmt.Printf("gettransaction: %v\n", err)
		return nil, ""
	}
	if getTransactionResult.Confirmations < 2 {
		fmt.Println("gettransaction less than 2 confs")
		return nil, ""
	}

	msgTx := wire.NewMsgTx()
	if err = msgTx.Deserialize(hex.NewDecoder(strings.NewReader(getTransactionResult.Hex))); err != nil {
		panic(err)
	}
	if len(msgTx.TxOut) < 2 {
		panic("msgTx.TxOut < 2")
	}

	const scriptVersion = 0
	_, submissionAddr, _, err := txscript.ExtractPkScriptAddrs(scriptVersion,
		msgTx.TxOut[0].PkScript, chaincfg.TestNet3Params())
	if err != nil {
		panic(err)
	}
	if len(submissionAddr) != 1 {
		panic(len(submissionAddr))
	}
	addr, err := stake.AddrFromSStxPkScrCommitment(msgTx.TxOut[1].PkScript,
		chaincfg.TestNet3Params())
	if err != nil {
		panic(err)
	}

	msg := fmt.Sprintf("vsp v3 getfeeaddress %s", msgTx.TxHash().String())
	var signature string
	err = c.Call(ctx, "signmessage", &signature, addr.Address(), msg)
	if err != nil {
		panic(err)
	}

	err = c.Call(ctx, "dumpprivkey", &privKeyStr, submissionAddr[0].Address())
	if err != nil {
		panic(err)
	}

	reqBytes, err := json.Marshal(GetFeeAddressRequest{
		TicketHash: msgTx.TxHash().String(),
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
	fmt.Printf("%+v\n", string(b))

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

func payFee(ctx context.Context, c *wsrpc.Client, privKeyWIF string, pubKey ed25519.PublicKey, address string, fee float64) error {
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
		Hex:       []byte(signedTx.Hex),
		VotingKey: privKeyWIF,
		Timestamp: time.Now().Unix(),
		VoteBits:  4,
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/payfee", bytes.NewBuffer(reqBytes))
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

	fmt.Printf("%+v\n", string(b))

	sigStr := resp.Header.Get("VSP-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		panic(err)
	}

	if !ed25519.Verify(pubKey, b, sig) {
		panic("bad signature")
	}

	var hash string
	err = c.Call(ctx, "sendrawtransaction", &hash, signedTx.Hex)
	if err != nil {
		fmt.Printf("failed to send tx: %v\n", err)
	}
	fmt.Printf("%s\n", hash)
	fmt.Printf("%s\n", string(b))
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

		err := payFee(ctx, c, privKeyStr, pubKey.PubKey, feeAddress.FeeAddress, fee.Fee)
		if err != nil {
			fmt.Printf("payFee: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
}
