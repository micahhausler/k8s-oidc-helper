package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	flag "github.com/ogier/pflag"
	yaml "gopkg.in/yaml.v2"
)

const Version = "0.0.1"

var version = flag.BoolP("version", "v", false, "print version and exit")

var openBrowser = flag.BoolP("open", "o", true, "Open the oauth approval URL in the browser")

var clientIDFlag = flag.String("client-id", "", "The ClientID for the application")
var clientSecretFlag = flag.String("client-secret", "", "The ClientSecret for the application")
var appFile = flag.StringP("config", "c", "", "Path to a json file containing your application's ClientID and ClientSecret. Supercedes the --client-id and --client-secret flags.")

const oauthUrl = "https://accounts.google.com/o/oauth2/auth?redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&client_id=%s&scope=openid+email+profile&approval_prompt=force&access_type=offline"

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

func readConfig(path string) (*GoogleConfig, error) {
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
func getTokens(clientID, clientSecret, code string) (*TokenResponse, error) {
	val := url.Values{}
	val.Add("grant_type", "authorization_code")
	val.Add("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	val.Add("client_id", clientID)
	val.Add("client_secret", clientSecret)
	val.Add("code", code)

	resp, err := http.PostForm("https://www.googleapis.com/oauth2/v3/token", val)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
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
}

func getUserEmail(accessToken string) (string, error) {
	uri, _ := url.Parse("https://www.googleapis.com/oauth2/v1/userinfo")
	q := uri.Query()
	q.Set("alt", "json")
	q.Set("access_token", accessToken)
	uri.RawQuery = q.Encode()
	resp, err := http.Get(uri.String())
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	ui := &UserInfo{}
	err = json.NewDecoder(resp.Body).Decode(ui)
	if err != nil {
		return "", err
	}
	return ui.Email, nil
}

func generateUser(email, clientId, clientSecret, idToken, refreshToken string) *KubectlUser {
	return &KubectlUser{
		Name: email,
		KubeUserInfo: &KubeUserInfo{
			AuthProvider: &AuthProvider{
				APConfig: &APConfig{
					ClientID:     clientId,
					ClientSecret: clientSecret,
					IdToken:      idToken,
					IdpIssuerUrl: "https://accounts.google.com",
					RefreshToken: refreshToken,
				},
				Name: "oidc",
			},
		},
	}
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Printf("k8s-oidc-helper %s\n", Version)
		os.Exit(0)
	}

	var gcf *GoogleConfig
	var err error
	if len(*appFile) > 0 {
		gcf, err = readConfig(*appFile)
		if err != nil {
			fmt.Printf("Error reading config file %s: %s\n", *appFile, err)
			os.Exit(1)
		}
	}
	var clientID string
	var clientSecret string
	if gcf != nil {
		clientID = gcf.ClientID
		clientSecret = gcf.ClientSecret
	} else {
		clientID = *clientIDFlag
		clientSecret = *clientSecretFlag
	}

	if *openBrowser {
		cmd := exec.Command("open", fmt.Sprintf(oauthUrl, clientID))
		err = cmd.Start()
	}
	if !*openBrowser || err != nil {
		fmt.Printf("Open this url in your browser: %s\n", fmt.Sprintf(oauthUrl, clientID))
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the code Google gave you: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	tokResponse, err := getTokens(clientID, clientSecret, code)
	if err != nil {
		fmt.Printf("Error getting tokens: %s\n", err)
		os.Exit(1)
	}

	email, err := getUserEmail(tokResponse.AccessToken)
	if err != nil {
		fmt.Printf("Error getting user email: %s\n", err)
		os.Exit(1)
	}

	userConfig := generateUser(email, clientID, clientSecret, tokResponse.IdToken, tokResponse.RefreshToken)
	output := map[string][]*KubectlUser{}
	output["users"] = []*KubectlUser{userConfig}
	response, err := yaml.Marshal(output)
	if err != nil {
		fmt.Printf("Error marshaling yaml: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("\n# Add the following to your ~/.kube/config")
	fmt.Println(string(response))

}
