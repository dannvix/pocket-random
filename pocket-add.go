package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"
	"time"
)

const (
	API_BASE = "https://getpocket.com/v3/"
)

////////////////////////////////////////////////////////////////////////////////

func truncateString(input string, width int) string {
	if len(input) > width {
		return input[:width-1] + "…"
	}
	return input
}

////////////////////////////////////////////////////////////////////////////////

type UserConfig struct {
	ApiKey    string `json:"api_key"`
	Username  string `json:"username"`
	UserCode  string `json:"user_code"`
	UserToken string `json:"user_token"`
	pathName  string
}

func NewUserConfig() *UserConfig {
	cfgFileName := ".pocketrandom"

	currentUser, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	cfg := new(UserConfig)
	cfg.pathName = path.Join(currentUser.HomeDir, cfgFileName)
	return cfg
}

func (cfg *UserConfig) save() {
	cfgJson, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(cfg.pathName, cfgJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (cfg *UserConfig) loadOrInitialize() *UserConfig {
	raw, err := ioutil.ReadFile(cfg.pathName)
	if err != nil {
		cfg.save()
	}

	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		cfg.save()
	}
	return cfg
}

////////////////////////////////////////////////////////////////////////////////

func requestPocketApi(cfg *UserConfig, requestPath string, requestData map[string]interface{}) map[string]interface{} {
	if cfg.ApiKey != "" {
		requestData["consumer_key"] = cfg.ApiKey
	}
	if cfg.UserToken != "" {
		requestData["access_token"] = cfg.UserToken
	}

	requestData["time"] = fmt.Sprintf("%d", int32(time.Now().Unix()))

	requestUrl := API_BASE + requestPath
	requestBody, err := json.Marshal(requestData)
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("X-Accept", "application/json")

	// Dump request content for debugging (requires `net/http/httputil`)
	// dump, err := httputil.DumpRequestOut(req, true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%s\n", dump)

	resp, err := http.DefaultClient.Do(req)
	if (err != nil) || (resp.StatusCode != 200) {
		log.Fatalf("POST '%s' failed, StatusCode=[%d], X-Error=[%s], X-Error-Code=[%s]",
			requestUrl, resp.StatusCode, resp.Header.Get("X-Error"), resp.Header.Get("X-Error-Code"))
	}
	defer resp.Body.Close()

	var bodyJson map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&bodyJson)
	if err != nil {
		log.Fatal(err)
	}
	return bodyJson
}

func requestApiKey(cfg *UserConfig) {
	pocketDevSite := "http://getpocket.com/developer/apps/"
	fmt.Printf("No API key available. Get one on %s\n", pocketDevSite)
	webbrowser.Open(pocketDevSite)

	fmt.Printf("Enter your API key: ")
	apiKey, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	cfg.ApiKey = strings.Trim(apiKey, " \t\n")
	cfg.save()
}

func requestUserCode(cfg *UserConfig) {
	const redirectUri = "https://getpocket.com/connected_accounts"
	requestData := map[string]interface{}{
		"redirect_uri": redirectUri,
	}

	body := requestPocketApi(cfg, "oauth/request", requestData)
	userCode := body["code"].(string)
	fmt.Printf("OAuth code: %s\n", userCode)

	authorizationUrl := fmt.Sprintf(
		"https://getpocket.com/auth/authorize?request_token=%s&redirect_uri=%s",
		userCode, redirectUri)
	fmt.Printf("Please authorize this app on %s\n", authorizationUrl)
	webbrowser.Open(authorizationUrl)

	fmt.Printf("Press any key to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	cfg.UserCode = userCode
	cfg.save()
}

func requestUserToken(cfg *UserConfig) {
	requestData := map[string]interface{}{
		"code": cfg.UserCode,
	}

	body := requestPocketApi(cfg, "oauth/authorize", requestData)
	userName := body["username"].(string)
	userToken := body["access_token"].(string)
	fmt.Printf("Username: %s\n", userName)
	fmt.Printf("Auth token: %s\n", userToken)

	cfg.Username = userName
	cfg.UserToken = userToken
	cfg.save()
}

func requestPermission(cfg *UserConfig) {
	if cfg.ApiKey == "" {
		requestApiKey(cfg)
	}

	if cfg.UserCode == "" {
		requestUserCode(cfg)
	}

	if cfg.UserToken == "" {
		requestUserToken(cfg)
	}
}

func addItemWithUrl(cfg *UserConfig, itemUrl string) map[string]interface{} {
	respChnl := make(chan map[string]interface{}, 0)

	go func() {
		requestData := map[string]interface{}{
			"url": itemUrl,
		}

		body := requestPocketApi(cfg, "add", requestData)
		respChnl <- body
	}()

	var body map[string]interface{}
	func() {
		for true {
			for _, slash := range "|/-\\" {
				fmt.Printf("Saving item to Pocket ... %c\r", slash)
				select {
				case body = <-respChnl:
					fmt.Printf("Saving item to Pocket ... done!\n")
					return
				case <-time.After(250 * time.Millisecond):
					continue
				}
			}
		}
	}()

	item := body["item"].(map[string]interface{})
	return item
}

////////////////////////////////////////////////////////////////////////////////

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <URL>>\n", os.Args[0])
	}
	itemUrl := os.Args[1]

	cfg := NewUserConfig().loadOrInitialize()
	requestPermission(cfg)

	if cfg.ApiKey == "" || cfg.UserCode == "" || cfg.UserToken == "" {
		log.Fatalf("Configuration error, api_key=[%s], user_code=[%s], user_token=[%s]",
			cfg.ApiKey, cfg.UserCode, cfg.UserToken)
	}
	fmt.Printf("Hello %s!\n", cfg.Username)

	termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default fallback
	}

	item := addItemWithUrl(cfg, itemUrl)

	itemId := item["item_id"].(string)
	itemResolvedUrl := item["resolved_normal_url"].(string)
	itemTitle := item["title"].(string)
	itemExcerpt := item["excerpt"].(string)

	writeStringWithPrefixAndTruncate := func(prefix string, arg string) {
		fmtStr := fmt.Sprintf("%s: %%s\n", prefix)
		fmt.Printf(fmtStr, truncateString(arg, termWidth-len(prefix)-len(": ")-1))
	}

	writeStringWithPrefixAndTruncate("Item ID", itemId)
	writeStringWithPrefixAndTruncate("Item URL", itemResolvedUrl)
	writeStringWithPrefixAndTruncate("Item Title", itemTitle)
	writeStringWithPrefixAndTruncate("Item Excerpt", itemExcerpt)
}
