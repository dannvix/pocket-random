package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	API_BASE = "https://getpocket.com/v3/"
)

////////////////////////////////////////////////////////////////////////////////

func prettyDateSince(unixTime int) string {
	then := time.Unix(int64(unixTime), 0)
	sinceThen := time.Since(then)
	diffDays := int(sinceThen.Hours() / 24)
	switch {
	case diffDays < 0:
		return "from the future!"
	case diffDays == 0:
		diffSeconds := int(sinceThen.Seconds())
		switch {
		case diffSeconds < 60:
			return "few seconds ago"
		case diffSeconds < 120:
			return "a minute ago"
		case diffSeconds < 3600:
			return fmt.Sprintf("%d minutes ago", int(diffSeconds/60))
		case diffSeconds < 7200:
			return "an hour ago"
		default:
			return fmt.Sprintf("%d hours ago", int(diffSeconds/3600))
		}
	case diffDays == 1:
		return "yesterday"
	case diffDays < 7:
		return fmt.Sprintf("%d days ago", int(diffDays))
	case diffDays < 31:
		return fmt.Sprintf("%d weeks ago", int(diffDays/7))
	case diffDays < 365:
		return fmt.Sprintf("%d months ago", int(diffDays/30))
	default:
		return fmt.Sprintf("%d years ago", int(diffDays/365))
	}
	return ""
}

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

func requestPocketApi(cfg *UserConfig, requestPath string, requestData url.Values) map[string]interface{} {
	if cfg.ApiKey != "" {
		requestData.Add("consumer_key", cfg.ApiKey)
	}
	if cfg.UserToken != "" {
		requestData.Add("access_token", cfg.UserToken)
	}

	requestUrl := API_BASE + requestPath
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(requestData.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if (err != nil) || (resp.StatusCode != 200) {
		log.Fatalf("POST '%s' failed, StatusCode=[%d], X-Error-Code=[%s]",
			requestUrl, resp.StatusCode, resp.Header.Get("X-Error-Code"))
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
	requestData := url.Values{
		"redirect_uri": {redirectUri},
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
	requestData := url.Values{
		"code": {cfg.UserCode},
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

func retrieveItems(cfg *UserConfig) []map[string]interface{} {
	respChnl := make(chan map[string]interface{}, 0)

	go func() {
		requestData := url.Values{
			"detailType": {"simple"},
		}
		body := requestPocketApi(cfg, "get", requestData)
		respChnl <- body
	}()

	// wait for items, and animate the waiting message
	var body map[string]interface{}
	func() {
		for true {
			for _, slash := range "|/-\\" {
				fmt.Printf("Retrieving items from Pocket ... %c\r", slash)
				select {
				case body = <-respChnl:
					return
				case <-time.After(250 * time.Millisecond):
					continue
				}
			}
		}
	}()

	var items []map[string]interface{}
	for _, item := range body["list"].(map[string]interface{}) {
		items = append(items, item.(map[string]interface{}))
	}
	fmt.Printf("Retrieving items from Pocket... %d items retrieved!\n", len(items))

	// shuffle by randomly swap items
	shuffleRounds := rand.Intn(10)
	for sr := 0; sr < shuffleRounds; sr++ {
		for i := range items {
			j := rand.Intn(len(items))
			items[i], items[j] = items[j], items[i]
		}
	}
	return items
}

func userInteractOnItem(cfg *UserConfig, item map[string]interface{}) {
	for true {
		fmt.Printf("Action[help(h), o, a, f, d, n, q] > ")
		userAction, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		userAction = strings.Trim(strings.ToLower(userAction), " \r\n")

		itemAction := func(action string) {
			actions, err := json.Marshal([]map[string]string{
				{"action": action, "item_id": item["item_id"].(string)},
			})
			if err != nil {
				log.Fatal(err)
			}

			requestData := url.Values{
				"actions": {string(actions)},
			}
			requestPocketApi(cfg, "send", requestData)
			fmt.Printf("Item #%s %sd :-)\n", item["item_id"], action)
		}

		switch userAction {
		case "h", "help":
			fmt.Println(`
help (h) — show this message
open (o) — open the item in web browser
archive (o) — archive the item
favorite (f) — favorite the item
delete (d) — delete the item
next (n) — move to next item
quit (q) — quit this program`)
		case "o", "open":
			webbrowser.Open(item["resolved_url"].(string))
		case "a", "archive":
			itemAction("archive")
			return
		case "f", "favorite":
			itemAction("favorite")
		case "d", "delete":
			itemAction("delete")
			return
		case "n", "next":
			return
		case "q", "quit":
			os.Exit(0)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func main() {
	cfg := NewUserConfig().loadOrInitialize()
	requestPermission(cfg)

	// retrieve and go through items
	if cfg.ApiKey == "" || cfg.UserCode == "" || cfg.UserToken == "" {
		log.Fatalf("Configuration error, api_key=[%s], user_code=[%s], user_token=[%s]",
			cfg.ApiKey, cfg.UserCode, cfg.UserToken)
	}

	fmt.Printf("Hello %s!\n", cfg.Username)

	items := retrieveItems(cfg)
	for i := range items {
		item := items[i]
		itemUnixTime, _ := strconv.Atoi(item["time_added"].(string))

		termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			termWidth = 80 // default fallback
		}

		var itemTitle string
		switch {
		case item["resolved_title"] != nil && item["resolved_title"] != "":
			itemTitle = item["resolved_title"].(string)
		case item["given_title"] != nil && item["given_title"] != "":
			itemTitle = item["given_title"].(string)
		}

		itemId := fmt.Sprintf("[#%s]", item["item_id"])
		itemTitle = fmt.Sprintf("\"%s\"", truncateString(itemTitle, termWidth-len("\"\"")-len(itemId)-1))
		itemUrl := truncateString(item["resolved_url"].(string), termWidth-1)
		itemDate := fmt.Sprintf("Added %s", prettyDateSince(itemUnixTime))
		itemWordCount := fmt.Sprintf("~ %s words", item["word_count"])

		// use `fmt.Fprintln(color.Output, …)` to support Windows
		fmt.Println()
		if item["favorite"] != "0" {
			fmt.Fprintln(color.Output, color.RedString("★ FAVORITED"))
		}
		fmt.Fprintln(color.Output, color.YellowString(itemId)+" "+color.WhiteString(itemTitle))
		fmt.Fprintln(color.Output, color.GreenString(strings.Replace(itemUrl, "%", "%%", -1))) // `itemUrl` may contain '%xx' which can be considered as a format string
		fmt.Fprintln(color.Output, color.CyanString(itemDate)+" | "+color.MagentaString(itemWordCount))

		userInteractOnItem(cfg, item)
	}
}
