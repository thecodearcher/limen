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
	if discoveryURL == "" {
		return
	}

	doc, err := fetchDiscoveryDocument(discoveryURL)
	if err != nil {
		panic("oauth-consentkeys: discovery fetch failed: " + err.Error())
	}
	if c.authorizationURL == "" {
		c.authorizationURL = doc.AuthorizationEndpoint
	}
	if c.tokenURL == "" {
		c.tokenURL = doc.TokenEndpoint
	}
	if c.userInfoURL == "" {
		c.userInfoURL = doc.UserinfoEndpoint
	}
}

func (c *config) validateEndpoints() {
	if c.authorizationURL == "" {
		panic("oauth-consentkeys: authorization URL is required (use WithAuthorizationURL)")
	}
	if c.tokenURL == "" {
		panic("oauth-consentkeys: token URL is required (use WithTokenURL)")
	}
	if c.userInfoURL == "" {
		panic("oauth-consentkeys: userinfo URL is required (use WithUserInfoURL)")
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
