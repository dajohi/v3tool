package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
	wallettypes "decred.org/dcrwallet/rpc/jsonrpc/types"
	"github.com/jrick/wsrpc/v2"
)

const (
	baseURL = "https://teststakepool.decred.org"

	rpcURL = "wss://localhost:19109/ws"
	rpcUser = "test"
	rpcPass = "test"
)

func getPubKey() *GetPubKeyResponse {
	resp, err := http.Get(baseURL + "/api/v3/getpubkey")
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
	resp, err := http.Get(baseURL + "/api/v3/getfee")
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
	q := url.Values{}
	var getTransactionResult wallettypes.GetTransactionResult
	err := c.Call(ctx, "gettransaction", &getTransactionResult, ticket, false)
	if err != nil {
		fmt.Printf("gettransaction: %v\n", err)
		return nil, ""
	}
	if getTransactionResult.Confirmations < 2 {
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

	msg := fmt.Sprintf("vsp v2 getfeeaddress %s", msgTx.TxHash().String())
	var signature string
	err = c.Call(ctx, "signmessage", &signature, addr.Address(), msg)
	if err != nil {
		panic(err)
	}

	err = c.Call(ctx, "dumpprivkey", &privKeyStr, submissionAddr[0].Address())
	if err != nil {
		panic(err)
	}

	q.Add("ticketHash", msgTx.TxHash().String())
	q.Add("signature", signature)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v3/getfeeaddress", nil)
	if err != nil {
		panic(err)
	}
	req.URL.RawQuery = q.Encode()

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
	fmt.Printf("%v\n", msgtxstr)

	var fundtxstr = struct {
		Hex string  `json:"hex"`
		Fee float64 `json:"fee"`
	}{}
	zero := int32(0)
	opt := wallettypes.FundRawTransactionOptions{
		ConfTarget: &zero,
	}
	err = c.Call(ctx, "fundrawtransaction", &fundtxstr, msgtxstr, "default", &opt)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", fundtxstr)

	var signedTxstr = struct {
		Hex      string `json:"hex"`
		Complete bool   `json:"complete"`
	}{}
	err = c.Call(ctx, "signrawtransaction", &signedTxstr, fundtxstr.Hex)
	if err != nil {
		return err
	}
	if !signedTxstr.Complete {
		return fmt.Errorf("not all signed")
	}
	values := url.Values{}
	values.Set("feeTx", signedTxstr.Hex)
	values.Set("votingKey", privKeyWIF)

	resp, err := http.PostForm(baseURL+"/api/v3/payfee", values)
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

	var hash string
	err = c.Call(ctx, "sendrawtransaction", &hash, signedTxstr.Hex)
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
	err = c.Call(ctx, "gettickets", &tickets, false)
	if err != nil {
		panic(err)
	}
	if len(tickets.Hashes) == 0 {
		panic("no tickets")
	}
	fmt.Printf("%d tickets", len(tickets.Hashes))
	for i := 0; i < len(tickets.Hashes); i++ {
		feeAddress, privKeyStr := getFeeAddress(ctx, c, pubKey.PubKey, tickets.Hashes[i])
		if feeAddress == nil {
			continue
		}
		fmt.Printf("feeAddress: %v\n", feeAddress.FeeAddress)

		err := payFee(ctx, c, privKeyStr, pubKey.PubKey, feeAddress.FeeAddress, fee.Fee)
		if err != nil {
			fmt.Printf("payFee: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
}
