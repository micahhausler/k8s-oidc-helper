package helper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ConfigFile struct {
	Installed *GoogleConfig `json:"installed"`
}

type GoogleConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

func ReadConfig(path string) (*GoogleConfig, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	cf := &ConfigFile{}
	err = json.NewDecoder(f).Decode(cf)
	if err != nil {
		return nil, err
	}
	return cf.Installed, nil
}

// Get the id_token and refresh_token from google
func GetToken(clientID, clientSecret, code string) (*TokenResponse, error) {
	val := url.Values{}
	val.Add("grant_type", "authorization_code")
	val.Add("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	val.Add("client_id", clientID)
	val.Add("client_secret", clientSecret)
	val.Add("code", code)

	resp, err := http.PostForm("https://www.googleapis.com/oauth2/v4/token", val)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tr := &TokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

type KubectlUser struct {
	Name         string        `yaml:"name"`
	KubeUserInfo *KubeUserInfo `yaml:"user"`
}

type KubeUserInfo struct {
	AuthProvider *AuthProvider `yaml:"auth-provider"`
}

type AuthProvider struct {
	APConfig *APConfig `yaml:"config"`
	Name     string    `yaml:"name"`
}

type APConfig struct {
	ClientID     string `yaml:"client-id"`
	ClientSecret string `yaml:"client-secret"`
	IDToken      string `yaml:"id-token"`
	IdpIssuerURL string `yaml:"idp-issuer-url"`
	RefreshToken string `yaml:"refresh-token"`
}

type UserInfo struct {
	Email string `json:"email"`
}

func GetUserEmail(accessToken string) (string, error) {
	uri, _ := url.Parse("https://www.googleapis.com/oauth2/v1/userinfo")
	q := uri.Query()
	q.Set("alt", "json")
	q.Set("access_token", accessToken)
	uri.RawQuery = q.Encode()
	resp, err := http.Get(uri.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ui := &UserInfo{}
	err = json.NewDecoder(resp.Body).Decode(ui)
	if err != nil {
		return "", err
	}
	return ui.Email, nil
}

func GenerateAuthInfo(clientID, clientSecret, idToken, refreshToken string) *clientcmdapi.AuthInfo {
	return &clientcmdapi.AuthInfo{
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: "oidc",
			Config: map[string]string{
				"client-id":      clientID,
				"client-secret":  clientSecret,
				"id-token":       idToken,
				"idp-issuer-url": "https://accounts.google.com",
				"refresh-token":  refreshToken,
			},
		},
	}
}

func createOpenCmd(oauthURL, clientID string) (*exec.Cmd, error) {
	url := fmt.Sprintf(oauthURL, clientID)

	switch os := runtime.GOOS; os {
	case "darwin":
		return exec.Command("open", url), nil
	case "linux":
		return exec.Command("xdg-open", url), nil
	}

	return nil, fmt.Errorf("Could not detect the open command for OS: %s", runtime.GOOS)
}

func LaunchBrowser(openBrowser bool, oauthURL, clientID string) {
	openInstructions := fmt.Sprintf("Open this url in your browser: %s\n", fmt.Sprintf(oauthURL, clientID))

	if !openBrowser {
		fmt.Print(openInstructions)
		return
	}

	cmd, err := createOpenCmd(oauthURL, clientID)
	if err != nil {
		fmt.Print(openInstructions)
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Print(openInstructions)
	}
}
