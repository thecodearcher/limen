package oauthconsentkeys

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type discoveryDocument struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

func (c *config) resolveDiscovery(discoveryURL string) {
	doc, err := fetchDiscoveryDocument(discoveryURL)
	if err != nil {
		panic("oauth-consentkeys: discovery fetch failed: " + err.Error())
	}
	c.authorizationURL = doc.AuthorizationEndpoint
	c.tokenURL = doc.TokenEndpoint
	c.userInfoURL = doc.UserinfoEndpoint
}

func (c *config) validateEndpoints() {
	if c.authorizationURL == "" || c.tokenURL == "" || c.userInfoURL == "" {
		panic("oauth-consentkeys: discovery document missing required endpoints")
	}
}

func fetchDiscoveryDocument(discoveryURL string) (*discoveryDocument, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery fetch: %s", resp.Status)
	}

	var doc discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("discovery decode: %w", err)
	}
	return &doc, nil
}
