package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/micahhausler/k8s-oidc-helper/internal/helper"
	flag "github.com/spf13/pflag"
	viper "github.com/spf13/viper"
	k8s_runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

const Version = "v0.1.1-alpha.1"

// urlTemplate is the template URL for the OIDC IDP request.
var urlTemplate = template.Must(template.New("url").Parse("{{.URLPath}}?redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&client_id={{.ClientID}}&scope={{.Scope}}&approval_prompt=force&access_type=offline"))

// UrlValues contains all values that will be substituted into the URL template.
type UrlValues struct {
	URLPath  string
	Scope    string
	ClientID string
}

func main() {
	flag.BoolP("version", "v", false, "Print version and exit")
	flag.BoolP("open", "o", true, "Open the oauth approval URL in the browser")
	flag.String("client-id", "", "The ClientID for the application")
	flag.String("client-secret", "", "The ClientSecret for the application")
	flag.StringP("config", "c", "", "Path to a json file containing your application's ClientID and ClientSecret. Supercedes the --client-id and --client-secret flags.")
	flag.BoolP("write", "w", false, "Write config to file. Merges in the specified file")
	flag.String("file", "", "The file to write to. If not specified, `~/.kube/config` is used")

	var urlTpl UrlValues
	flag.StringVar(&urlTpl.URLPath, "oauth-url", "https://accounts.google.com/o/oauth2/auth", "The identity provider URL")
	flag.StringVar(&urlTpl.Scope, "scope", "openid+email+profile", "The scope to request from the identity provider")

	viper.BindPFlags(flag.CommandLine)
	viper.SetEnvPrefix("k8s-oidc-helper")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	flag.Parse()

	urlTpl.Scope = url.PathEscape(urlTpl.Scope)

	if viper.GetBool("version") {
		fmt.Printf("k8s-oidc-helper %s\n", Version)
		os.Exit(0)
	}

	var gcf *helper.GoogleConfig
	var err error
	if configFile := viper.GetString("config"); len(viper.GetString("config")) > 0 {
		gcf, err = helper.ReadConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config file %s: %s\n", configFile, err)
			os.Exit(1)
		}
	}

	var clientSecret string
	if gcf != nil {
		urlTpl.ClientID = gcf.ClientID
		clientSecret = gcf.ClientSecret
	} else {
		urlTpl.ClientID = viper.GetString("client-id")
		clientSecret = viper.GetString("client-secret")
	}

	url := bytes.NewBuffer(nil)
	if err := urlTemplate.Execute(url, urlTpl); err != nil {
		fmt.Printf("Error building request URL: %s\n", err)
		os.Exit(2)
	}

	helper.LaunchBrowser(viper.GetBool("open"), url.String())

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the code Google gave you: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	tokResponse, err := helper.GetToken(urlTpl.ClientID, clientSecret, code)
	if err != nil {
		fmt.Printf("Error getting tokens: %s\n", err)
		os.Exit(1)
	}

	email, err := helper.GetUserEmail(tokResponse.AccessToken)
	if err != nil {
		fmt.Printf("Error getting user email: %s\n", err)
		os.Exit(1)
	}

	authInfo := helper.GenerateAuthInfo(urlTpl.ClientID, clientSecret, tokResponse.IdToken, tokResponse.RefreshToken)
	config := &clientcmdapi.Config{
		AuthInfos: map[string]*clientcmdapi.AuthInfo{email: authInfo},
	}

	if !viper.GetBool("write") {
		fmt.Println("\n# Add the following to your ~/.kube/config")

		json, err := k8s_runtime.Encode(clientcmdlatest.Codec, config)
		if err != nil {
			fmt.Printf("Unexpected error: %v", err)
			os.Exit(1)
		}
		output, err := yaml.JSONToYAML(json)
		if err != nil {
			fmt.Printf("Unexpected error: %v", err)
			os.Exit(1)
		}
		fmt.Printf("%v", string(output))
		return
	}

	tempKubeConfig, err := ioutil.TempFile("", "")
	if err != nil {
		fmt.Printf("Could not create tempfile: %v", err)
		os.Exit(1)
	}
	defer os.Remove(tempKubeConfig.Name())
	clientcmd.WriteToFile(*config, tempKubeConfig.Name())

	var kubeConfigPath string
	if viper.GetString("file") == "" {
		usr, err := user.Current()
		if err != nil {
			fmt.Printf("Could not determine current: %v", err)
			os.Exit(1)
		}
		kubeConfigPath = filepath.Join(usr.HomeDir, ".kube", "config")
	} else {
		kubeConfigPath = viper.GetString("file")
	}

	loadingRules := clientcmd.ClientConfigLoadingRules{
		Precedence: []string{tempKubeConfig.Name(), kubeConfigPath},
	}
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		fmt.Printf("Could not merge configuration: %v", err)
		os.Exit(1)
	}

	clientcmd.WriteToFile(*mergedConfig, kubeConfigPath)
	fmt.Printf("Configuration has been written to %s\n", kubeConfigPath)
}
