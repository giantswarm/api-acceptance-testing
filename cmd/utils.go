package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/fatih/color"
	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/microerror"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

func getClientAuth(scheme string, token string) (runtime.ClientAuthInfoWriter, error) {
	if scheme == "" {
		return nil, microerror.New("Invalid scheme provided")
	}
	if token == "" {
		return nil, microerror.New("Invalid token provided")
	}

	authHeader := scheme + " " + token
	return httptransport.APIKeyAuth("Authorization", "header", authHeader), nil
}

func newClient(endpointURL string) (*gsclient.Gsclientgen, error) {
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

// exitIfError handles an error of it gets one, by printing it
// and exiting with status code 1.
func exitIfError(err error) {
	if err == nil {
		return
	}
	complain(err)
	os.Exit(1)
}

// complainIfError handles an error of it gets one, by printing it
// but does not exit.
func complain(err error) {
	if err == nil {
		return
	}

	fmt.Printf("%s: %s\n", color.RedString("ERROR"), color.WhiteString(err.Error()))
}

func printSuccess(message string, v ...interface{}) {
	fmt.Printf("%s: %s\n", color.GreenString("OK"), color.WhiteString(message, v...))
}
