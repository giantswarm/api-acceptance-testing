package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/giantswarm/gscliauth/oidc"
	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/microerror"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// Client is our API client.
type Client struct {
	APIEndpointURL string

	IDToken      *oidc.IDToken
	AccessToken  string
	RefreshToken string

	GSClientGen *gsclient.Gsclientgen
}

// New returns a fully conifgured API client and initiates the browser auth flow.
func New(endpointURL string) (*Client, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, microerror.Maskf(invalidConfigError, "invalid endpoint URL")
	}

	tlsConfig := &tls.Config{}

	transport := httptransport.New(u.Host, "", []string{u.Scheme})
	transport.Transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}

	c := &Client{
		APIEndpointURL: endpointURL,
		GSClientGen:    gsclient.New(transport, strfmt.Default),
	}

	pkceResponse, err := oidc.RunPKCE(endpointURL)
	if err != nil {
		fmt.Println("DEBUG: Attempt to run the OAuth2 PKCE workflow with a local callback HTTP server failed.")
		return nil, microerror.Mask(err)
	}

	idToken, err := oidc.ParseIDToken(pkceResponse.IDToken)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// store tokens
	c.IDToken = idToken
	c.AccessToken = pkceResponse.AccessToken
	c.RefreshToken = pkceResponse.RefreshToken

	return c, nil
}

// AuthHeaderWriter returns a function to write an authentication header.
func (c *Client) AuthHeaderWriter() (runtime.ClientAuthInfoWriter, error) {
	authHeader := "Bearer " + c.MustGetToken()
	return httptransport.APIKeyAuth("Authorization", "header", authHeader), nil
}

// GetToken returns a token to use for the API client's Authorization header.
// If necessary, refreshes the token.
func (c *Client) GetToken() (string, error) {
	// Check if it has a refresh token.
	if c.RefreshToken == "" {
		return "", microerror.Maskf(invalidConfigError, "No refresh token saved in config file, unable to acquire new access token. Please login again.")
	}

	if isTokenExpired(c.AccessToken) {
		// Get a new token.
		refreshTokenResponse, err := oidc.RefreshToken(c.RefreshToken)
		if err != nil {
			return "", microerror.Mask(err)
		}

		// Parse the ID Token for the email address.
		idToken, err := oidc.ParseIDToken(refreshTokenResponse.IDToken)
		if err != nil {
			return "", microerror.Mask(err)
		}

		// Update the config file with the new access token.
		c.IDToken = idToken
		c.AccessToken = refreshTokenResponse.AccessToken

		fmt.Println("DEBUG: Access token has just been refreshed.")
	}

	return c.AccessToken, nil
}

// MustGetToken returns a token and only prints errors.
func (c *Client) MustGetToken() string {
	token, err := c.GetToken()
	if err != nil {
		fmt.Printf("ERROR: Could not get auth token. %s\n", err.Error())
	}

	return token
}

// isTokenExpired checks whwther this token is expired.
func isTokenExpired(token string) bool {
	// Parse token
	claims := jwtgo.MapClaims{}

	parsedToken, _, err := new(jwtgo.Parser).ParseUnverified(token, claims)
	if err != nil {
		return true
	}

	err = parsedToken.Claims.Valid()
	if err != nil {
		return true
	}

	return false
}
