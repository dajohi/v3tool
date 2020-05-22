package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// signedHTTP makes a request against a VSP API. The request will be JSON
// encoded and signed using the provided commitment address. The signature of
// the response is also validated using the VSPs pubkey.
func signedHTTP(url, method, commitmentAddr string, request interface{}) ([]byte, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, baseURL+url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}

	signature, err := signMessage(commitmentAddr, reqBytes)
	if err != nil {
		return nil, err
	}

	req.Header.Add("VSP-Client-Signature", signature)

	var httpClient http.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n%s response: %+v\n", url, string(b))

	sigStr := resp.Header.Get("VSP-Server-Signature")
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		return nil, fmt.Errorf("Error validating VSP signature: %v", err)
	}

	if !ed25519.Verify(vspPubKey, b, sig) {
		return nil, errors.New("Bad signature from VSP")
	}

	return b, nil
}
