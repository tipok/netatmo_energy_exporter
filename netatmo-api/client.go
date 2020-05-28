package netatmo_api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

const (
	authURL  = "https://api.netatmo.com/oauth2/authorize"
	tokenURL = "https://api.netatmo.com/oauth2/token"
)

// Config contains configuration for OAuth2
type Config struct {
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// Client working with netatmo API
type Client struct {
	httpClient *http.Client
	ctx        context.Context
}

// NewClient creates a new authenticated client
func NewClient(ctx context.Context, cnf *Config) (*Client, error) {

	httpClient, err := getOauthClient(ctx, cnf)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: httpClient,
		ctx:        ctx,
	}, nil
}

func getOauthClient(ctx context.Context, cnf *Config) (*http.Client, error) {
	oauth := &oauth2.Config{
		ClientID:     cnf.ClientID,
		ClientSecret: cnf.ClientSecret,
		Scopes:       cnf.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	token, err := oauth.PasswordCredentialsToken(ctx, cnf.Username, cnf.Password)
	if err != nil {
		return nil, fmt.Errorf("could not get token for %v: %w", cnf.Username, err)
	}

	httpClient := oauth.Client(ctx, token)

	return httpClient, nil
}

func closeBody(res *http.Response) {
	err := res.Body.Close()
	if err != nil {
		log.Printf("Error during body close: %v\n", err)
	}
}

func (c *Client) get(u *url.URL, v interface{}) error {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	return c.request(req, v)
}

func (c *Client) request(req *http.Request, v interface{}) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error during http request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		var objmap map[string]json.RawMessage
		if err := json.NewDecoder(res.Body).Decode(&objmap); err != nil {
			return fmt.Errorf("could not decode json: %w", err)
		}
		if body, ok := objmap["body"]; ok {
			if err := json.Unmarshal(body, &v); err != nil {
				return fmt.Errorf("could not decode body: %w", err)
			}
			return nil
		}
		return fmt.Errorf("could not find body: %v", objmap)
	default:
		bodyString, _ := readString(res)
		return fmt.Errorf("invalid request: status_code = %d content=%v", res.StatusCode, bodyString)
	}
}

func readString(resp *http.Response) (string, error) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyString := string(bodyBytes)
	return bodyString, nil
}
