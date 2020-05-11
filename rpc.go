package main

import (
	"context"
	"crypto/tls"

	"github.com/jrick/wsrpc/v2"
)

func NewRPC(ctx context.Context, rpcURL, rpcUser, rpcPass string) (*wsrpc.Client, error) {
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}
	tlsOpt := wsrpc.WithTLSConfig(&tlsConfig)
	authOpt := wsrpc.WithBasicAuth(rpcUser, rpcPass)

	return wsrpc.Dial(ctx, rpcURL, tlsOpt, authOpt)
}
