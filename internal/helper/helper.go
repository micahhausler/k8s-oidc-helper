package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

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
	IdToken      string `json:"id_token"`
}

type DiscoverySpec struct {
	AuthorizationEndpoint  string   `json:"authorization_endpoint"`
	TokenEndpoint          string   `json:"token_endpoint"`
	ScopesSupported        []string `json:"scopes_supported"`
	ResponseTypesSupported []string `json:"response_types_supported"`
	UserinfoEndpoint       string   `json:"userinfo_endpoint"`
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
func GetToken(ds DiscoverySpec, clientID, clientSecret, code string, redirectUri string) (*TokenResponse, error) {
	val := url.Values{}
	val.Add("grant_type", "authorization_code")
	val.Add("redirect_uri", redirectUri)
	val.Add("client_id", clientID)
	val.Add("client_secret", clientSecret)
	val.Add("code", code)

	resp, err := http.PostForm(ds.TokenEndpoint, val)
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
	IdToken      string `yaml:"id-token"`
	IdpIssuerUrl string `yaml:"idp-issuer-url"`
	RefreshToken string `yaml:"refresh-token"`
}

type UserInfo struct {
	Email string `json:"email"`
	Sub   string `json:"sub"`
	Name  string `json:"name"`
}

func GetUserClaim(ds DiscoverySpec, accessToken, userClaim string) (string, error) {
	uri, _ := url.Parse(ds.UserinfoEndpoint)
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
	var retVal string
	switch userClaim {
	case "email":
		retVal = ui.Email
	case "sub":
		retVal = ui.Sub
	case "name":
		retVal = ui.Name
	default:
		return "", errors.New("User Claim needs to be on of sub/name/email")
	}

	if retVal == "" {
		return "", fmt.Errorf("UserInfo Endpoint does not support provided claim: %s", userClaim)
	} else {
		return retVal, nil
	}
}

func GenerateAuthInfo(issuer, clientId, clientSecret, idToken, refreshToken string) *clientcmdapi.AuthInfo {
	return &clientcmdapi.AuthInfo{
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: "oidc",
			Config: map[string]string{
				"client-id":      clientId,
				"client-secret":  clientSecret,
				"id-token":       idToken,
				"idp-issuer-url": issuer,
				"refresh-token":  refreshToken,
			},
		},
	}
}

func createOpenCmd(oauthUrl string) (*exec.Cmd, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return exec.Command("open", oauthUrl), nil
	case "linux":
		return exec.Command("xdg-open", oauthUrl), nil
	}
	return nil, fmt.Errorf("Could not detect the open command for OS: %s", runtime.GOOS)
}

func LaunchBrowser(openBrowser bool, oauthUrl string) {
	openInstructions := fmt.Sprintf("Open this url in your browser: %s\n", oauthUrl)

	if !openBrowser {
		fmt.Print(openInstructions)
		return
	}

	cmd, err := createOpenCmd(oauthUrl)
	if err != nil {
		fmt.Print(openInstructions)
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Print(openInstructions)
	}
}

func ConstructAuthUrl(discoverySpec DiscoverySpec, scopes string, redirectUri string, clientID string) string {
	authURL, _ := url.Parse(discoverySpec.AuthorizationEndpoint)
	q := authURL.Query()
	// Some providers like Google accept a diiferent Query Parameter called access_type, Some Like Auth0 support it as a scope value, And Some like Gitlab Always give refresh tokens
	if contains(discoverySpec.ScopesSupported, "offline_access") {
		scopes = strings.TrimSpace(scopes) + " offline_access"
	}
	q.Set("scope", scopes)
	q.Set("redirect_uri", redirectUri)
	//TODO: check whether response_type is supported and throw error accordingly, but almost all the providers support code method
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("approval_prompt", "force") // Providers who dont support it should ignore any extra parameters
	q.Set("access_type", "offline")   // Providers who dont support it should ignore any extra parameters

	authURL.RawQuery = q.Encode()
	return authURL.String()
}

func GetDiscoverySpec(issuer string) (DiscoverySpec, error) {
	ds := &DiscoverySpec{}
	resp, err := http.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		return *ds, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(ds)
	if err != nil {
		return *ds, err
	}
	return *ds, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.Compare(a, e) == 0 {
			return true
		}
	}
	return false
}
