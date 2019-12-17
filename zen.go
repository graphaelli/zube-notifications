package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type client struct {
	clientId   string
	key        *rsa.PrivateKey
	httpClient *http.Client

	accessDuration time.Duration
	apiBaseUrl     string
	debug          bool

	accessMutex  sync.Mutex
	accessExpiry time.Time
	accessToken  string
}

type option func(*client)

func HttpClientOption(httpClient *http.Client) option {
	return func(c *client) {
		c.httpClient = httpClient
	}
}

func DebugOption(enabled bool) option {
	return func(c *client) {
		c.debug = enabled
	}
}

func NewClient(clientId string, key *rsa.PrivateKey, options ...option) *client {
	c := &client{
		clientId: clientId,
		key:      key,

		accessDuration: 1 * time.Minute,
		apiBaseUrl:     "https://zube.io/api/",
	}
	for _, o := range options {
		o(c)
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

func (c *client) refreshToken(iat, eat time.Time) (string, error) {
	now := time.Now()
	claims := &jwt.StandardClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: (now.Add(time.Minute)).Unix(),
		Issuer:    c.clientId,
	}
	if err := claims.Valid(); err != nil {
		log.Fatalf("invalid claims: %s", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(c.key)
}

func (c *client) newRequest(method, api string, body io.Reader) (*http.Request, error) {
	// TODO: urljoin
	req, err := http.NewRequest(method, c.apiBaseUrl+api, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Client-ID", c.clientId)
	return req, nil
}

func (c *client) doRequest(req *http.Request) (*http.Response, error) {
	if c.debug {
		log.Printf("doing %s %s", req.Method, req.URL.String())
	}
	if req.Header.Get("Authorization") == "" {
		c.accessMutex.Lock()
		if c.accessToken == "" || c.accessExpiry.Before(time.Now()) {
			now := time.Now()
			later := now.Add(c.accessDuration)

			accessToken, err := c.access(now, later)
			if err != nil {
				return nil, err
			}
			c.accessExpiry = later
			c.accessToken = accessToken
		}
		c.accessMutex.Unlock()
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	}
	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return rsp, err
	}
	if rsp.StatusCode == http.StatusBadRequest {
		body, _ := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()
		return rsp, fmt.Errorf("bad request: %s", string(body))
	}

	// dump response bodies to stdout while preserving rsp.Body.Close
	if c.debug {
		rsp.Body = struct {
			io.Reader
			io.Closer
		}{io.TeeReader(rsp.Body, log.Writer()), rsp.Body}
	}
	return rsp, err
}

func (c *client) access(issueTime, expireTime time.Time) (string, error) {
	refreshToken, err := c.refreshToken(issueTime, expireTime)
	if err != nil {
		return "", err
	}
	req, err := c.newRequest(http.MethodPost, "users/tokens", nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+refreshToken)
	rsp, err := c.doRequest(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()
	var accessTokenRsp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&accessTokenRsp); err != nil {
		return "", err
	}
	return accessTokenRsp.AccessToken, nil
}

func enabled(m map[string]interface{}) []string {
	var matches []string
	for k, v := range m {
		if b, ok := v.(bool); ok && b {
			matches = append(matches, k)
		}
	}
	return matches
}

func disableAll(m map[string]interface{}) {
	for k, v := range m {
		if b, ok := v.(bool); ok && b {
			m[k] = false
		}
	}
}

func main() {
	clientId := flag.String("c", os.Getenv("ZUBE_CLIENT_ID"), "zube client id")
	privateKeyFile := flag.String("k", "zube_api_key.pem", "path to zube api key pem")
	disableEmail := flag.Bool("E", false, "disable email notifications")
	disableInApp := flag.Bool("I", false, "disable in-app notifications")
	debug := flag.Bool("D", false, "enable debugging output")

	flag.Parse()
	if len(flag.Args()) > 1 {
		*clientId = flag.Arg(0)
	}
	if *clientId == "" {
		log.Fatal("client id required, set ZUBE_CLIENT_ID or provide as first argument")
	}

	privateKey, err := ioutil.ReadFile(*privateKeyFile)
	if err != nil {
		log.Fatal(err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	client := NewClient(*clientId, key, DebugOption(*debug))
	projects, err := client.ListProjects()
	if err != nil {
		log.Fatal(err)
	}
	for _, project := range projects {
		projectEmailPrefs, err := client.ProjectEmailPreferences(project.ID)
		if err != nil {
			log.Fatal(err)
		}
		projectInAppPrefs, err := client.ProjectInAppPreferences(project.ID)
		if err != nil {
			log.Fatal(err)
		}
		projectUserSettings, err := client.ProjectUserSettings(project.ID)
		if err != nil {
			log.Fatal(err)
		}
		projectTriageUserSettings, err := client.ProjectTriageUserSettings(project.ID)
		if err != nil {
			log.Fatal(err)
		}
		projectEmailNotifying := enabled(projectEmailPrefs)
		projectInAppNotifying := enabled(projectInAppPrefs)

		fmt.Printf("\n*** %s email: %s project: %s triage: %s, notifying: %d (email: %d in-app: %d)\n",
			project.Name,
			projectEmailPrefs["email"],
			projectUserSettings.SubscriptionLevel,
			projectTriageUserSettings.SubscriptionLevel,
			len(projectEmailNotifying)+len(projectInAppNotifying),
			len(projectEmailNotifying),
			len(projectInAppNotifying),
		)
		if *disableEmail {
			disableAll(projectEmailPrefs)
			projectPayloadEmail := new(bytes.Buffer)
			if err := json.NewEncoder(projectPayloadEmail).Encode(projectEmailPrefs); err != nil {
				log.Fatal(err)
			}
			id := int(projectEmailPrefs["id"].(float64))
			if err := client.DisableProjectEmailNotifications(project.ID, id, projectPayloadEmail); err != nil {
				log.Fatal(err)
			}
		}
		if *disableInApp {
			disableAll(projectInAppPrefs)
			projectPayloadInApp := new(bytes.Buffer)
			if err := json.NewEncoder(projectPayloadInApp).Encode(projectInAppPrefs); err != nil {
				log.Fatal(err)
			}
			id := int(projectInAppPrefs["id"].(float64))
			if err := client.DisableProjectInAppNotifications(project.ID, id, projectPayloadInApp); err != nil {
				log.Fatal(err)
			}
		}

		var wg sync.WaitGroup
		wg.Add(len(project.Workspaces))
		for _, w := range project.Workspaces {
			go func(workspace Workspace) {
				defer wg.Done()
				workspaceEmailPrefs, err := client.WorkspaceEmailPreferences(workspace.ID)
				if err != nil {
					log.Fatal(err)
				}
				workspaceInAppPrefs, err := client.WorkspaceInAppPreferences(workspace.ID)
				if err != nil {
					log.Fatal(err)
				}
				workspaceUserSettings, err := client.WorkspaceUserSettings(workspace.ID)
				if err != nil {
					log.Fatal(err)
				}
				workspaceEmailNotifying := enabled(workspaceEmailPrefs)
				workspaceInAppNotifying := enabled(workspaceInAppPrefs)

				fmt.Printf("\t%s email: %s project: %s triage: %s, notifying: %d (email: %d, in-app: %d)\n",
					workspace.Name,
					workspaceEmailPrefs["email"],
					workspaceUserSettings.SubscriptionLevel,
					workspaceUserSettings.SubscriptionLevel,
					len(workspaceEmailNotifying)+len(workspaceInAppNotifying),
					len(workspaceEmailNotifying),
					len(workspaceInAppNotifying),
				)

				if *disableEmail {
					disableAll(workspaceEmailPrefs)
					payloadEmail := new(bytes.Buffer)
					if err := json.NewEncoder(payloadEmail).Encode(workspaceEmailPrefs); err != nil {
						log.Fatal(err)
					}
					id := int(workspaceEmailPrefs["id"].(float64))
					if err := client.DisableWorkspaceEmailNotifications(workspace.ID, id, payloadEmail); err != nil {
						log.Fatal(err)
					}
				}
				if *disableInApp {
					disableAll(workspaceInAppPrefs)
					payloadInApp := new(bytes.Buffer)
					if err := json.NewEncoder(payloadInApp).Encode(workspaceInAppPrefs); err != nil {
						log.Fatal(err)
					}
					id := int(workspaceInAppPrefs["id"].(float64))
					if err := client.DisableWorkspaceInAppNotifications(workspace.ID, id, payloadInApp); err != nil {
						log.Fatal(err)
					}
				}
			}(w)
		}
		wg.Wait()
	}
}
