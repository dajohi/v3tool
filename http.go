package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// signedHTTP makes a request against a VSP API. The request will be JSON
// encoded and signed using the provided commitment address. The signature of
// the response is also validated using the VSPs pubkey.
func signedHTTP(url, method, commitmentAddr string, vspPubKey []byte, request interface{}) ([]byte, error) {
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Non 200 response from server: %v", string(b))
	}

	err = validateServerSignature(resp, b, vspPubKey)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func validateServerSignature(resp *http.Response, body []byte, pubKey []byte) error {
	sigStr := resp.Header.Get("VSP-Server-Signature")
	sig, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		return fmt.Errorf("Error validating VSP signature: %v", err)
	}

	if !ed25519.Verify(pubKey, body, sig) {
		return errors.New("Bad signature from VSP")
	}

	return nil
}
