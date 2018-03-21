package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/micahhausler/k8s-oidc-helper/internal/helper"
	flag "github.com/spf13/pflag"
	viper "github.com/spf13/viper"
	k8s_runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

const Version = "v0.2.0"

func main() {
	flag.BoolP("version", "v", false, "Print version and exit")
	flag.BoolP("open", "o", true, "Open the oauth approval URL in the browser")
	flag.String("client-id", "", "The ClientID for the application")
	flag.String("client-secret", "", "The ClientSecret for the application")
	flag.StringP("config", "c", "", "Path to a json file containing your Google application's ClientID and ClientSecret. Supercedes the --client-id and --client-secret flags.")
	flag.BoolP("write", "w", false, "Write config to file. Merges in the specified file")
	flag.String("file", "", "The file to write to. If not specified, `~/.kube/config` is used")
	flag.String("issuer-url", "", "OIDC Discovery URL, such that <URL>/.well-known/openid-configuration can be fetched")
	flag.String("scopes", "openid email", "Required scopes to be passed to the Authicator. offline_access is added if access_type parameter is not supported by authorizer")
	flag.String("redirect_uri", "http://localhost", "http://localhost or urn:ietf:wg:oauth:2.0:oob if --config flag is used for google OpenID")
	flag.String("user-claim", "email", "The Claim in ID-Token used to identify the user. One of sub/email/name")

	viper.BindPFlags(flag.CommandLine)
	viper.SetEnvPrefix("k8s-oidc-helper")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	flag.Parse()

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

	var clientID string
	var clientSecret string
	if gcf != nil {
		clientID = gcf.ClientID
		clientSecret = gcf.ClientSecret
	} else {
		clientID = viper.GetString("client-id")
		clientSecret = viper.GetString("client-secret")
	}

	var issuerUrl string
	redirectUri := viper.GetString("redirect_uri")
	if viper.GetString("issuer-url") == "" {
		issuerUrl = "https://accounts.google.com"
	} else {
		issuerUrl = viper.GetString("issuer-url")
	}

	if strings.HasPrefix(issuerUrl, "https://accounts.google.com") {
		redirectUri = "urn:ietf:wg:oauth:2.0:oob"
	}

	ds, err := helper.GetDiscoverySpec(issuerUrl)
	if err != nil {
		log.Fatalf("Can not get Discovery Spec, Please make sure that <URL>/.well-known/openid-configuration return OpenID JSON: %v", err)
	}

	helper.LaunchBrowser(viper.GetBool("open"), helper.ConstructAuthUrl(ds, viper.GetString("scopes"), redirectUri, clientID))

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the code Provider gave you (On The page or the Value of `code` query parameter on localhost URL) : ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	tokResponse, err := helper.GetToken(ds, clientID, clientSecret, code, redirectUri)
	if err != nil {
		fmt.Printf("Error getting tokens: %s\n", err)
		os.Exit(1)
	}

	email, err := helper.GetUserClaim(ds, tokResponse.AccessToken, viper.GetString("user-claim"))
	if err != nil {
		fmt.Printf("Error getting user email: %s\n", err)
		os.Exit(1)
	}

	authInfo := helper.GenerateAuthInfo(issuerUrl, clientID, clientSecret, tokResponse.IdToken, tokResponse.RefreshToken)
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
