package client

import (
	"crypto/tls"
	"net/http"
	"net/url"

	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/microerror"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

func ClientAuth(scheme string, token string) (runtime.ClientAuthInfoWriter, error) {
	if scheme == "" {
		return nil, microerror.New("Invalid scheme provided")
	}
	if token == "" {
		return nil, microerror.New("Invalid token provided")
	}

	authHeader := scheme + " " + token
	return httptransport.APIKeyAuth("Authorization", "header", authHeader), nil
}

func NewClient(endpointURL string) (*gsclient.Gsclientgen, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, microerror.New("invalid endpoint URL")
	}

	tlsConfig := &tls.Config{}

	transport := httptransport.New(u.Host, "", []string{u.Scheme})
	transport.Transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}

	return gsclient.New(transport, strfmt.Default), nil
}
