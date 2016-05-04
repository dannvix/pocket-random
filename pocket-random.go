package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/toqueteos/webbrowser"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
)

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
		os.Exit(1)
	}

	cfg := new(UserConfig)
	cfg.pathName = path.Join(currentUser.HomeDir, cfgFileName)
	return cfg
}

func (cfg UserConfig) save() {
	cfgJson, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(cfg.pathName, cfgJson, 0644)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func (cfg UserConfig) loadOrCreateNew() *UserConfig {
	raw, err := ioutil.ReadFile(cfg.pathName)
	if err != nil {
		cfg.save()
	}

	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		cfg.save()
	}
	return &cfg
}

func main() {
	const API_BASE = "https://getpocket.com/v3/"

	// ------------------
	// read configuration
	// ------------------
	cfg := NewUserConfig()
	cfg = cfg.loadOrCreateNew()

	// ---------------
	// request API key
	// ---------------
	if cfg.ApiKey == "" {
		pocketDevSite := "http://getpocket.com/developer/apps/"
		fmt.Printf("No API key available. Get one on %s\n", pocketDevSite)
		webbrowser.Open(pocketDevSite)

		fmt.Printf("Enter your API key: ")
		consoleReader := bufio.NewReader(os.Stdin)
		apiKey, err := consoleReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		cfg.ApiKey = strings.Trim(apiKey, " \t\n")
		cfg.save()
	}

	// -------------------------------------
	// request OAuth code (open web browser)
	// -------------------------------------
	if cfg.UserCode == "" {
		// FIXME: refactor duplicated HTTP communication code
		redirectUri := "https://getpocket.com/connected_accounts"
		requestUrl := fmt.Sprintf("%soauth/request", API_BASE)
		requestData := url.Values{
			"consumer_key": {cfg.ApiKey},
			"redirect_uri": {redirectUri},
		}

		req, err := http.NewRequest("POST", requestUrl, strings.NewReader(requestData.Encode()))
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(fmt.Sprintf("Request code failed, StatusCode=[%d], X-Error-Code=[%s]",
				resp.StatusCode, resp.Header.Get("X-Error-Code")))
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		userCode := data["code"].(string)
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

	// -------------------
	// request OAuth token
	// -------------------
	if cfg.UserToken == "" {
		// FIXME: refactor duplicated HTTP communication code
		requestUrl := fmt.Sprintf("%soauth/authorize", API_BASE)
		requestData := url.Values{
			"consumer_key": {cfg.ApiKey},
			"code":         {cfg.UserCode},
		}

		req, err := http.NewRequest("POST", requestUrl, strings.NewReader(requestData.Encode()))
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(fmt.Sprintf("Request authorization failed, StatusCode=[%d], X-Error-Code=[%s]",
				resp.StatusCode, resp.Header.Get("X-Error-Code")))
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		userName := data["username"].(string)
		userToken := data["access_token"].(string)
		fmt.Printf("Username: %s\n", userName)
		fmt.Printf("Auth token: %s\n", userToken)

		cfg.Username = userName
		cfg.UserToken = userToken
		cfg.save()
	}

	// -----------------------------
	// retrieve and go through items
	// -----------------------------
	if cfg.ApiKey != "" && cfg.UserCode != "" && cfg.UserToken != "" {
		fmt.Printf("Hello %s!\n", cfg.Username)

		request_url := fmt.Sprintf("%sget", API_BASE)
		// request_data := map[string]interface{} {
		request_data := url.Values{
			"consumer_key": {cfg.ApiKey},
			"access_token": {cfg.UserToken},
			"detailType":   {"simple"},
			"count":        {"10"},
		}

		fmt.Printf("Retrieving items from Pocket... ")

		resp, err := http.PostForm(request_url, request_data)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		var items []map[string]interface{}
		for _, item := range data["list"].(map[string]interface{}) {
			items = append(items, item.(map[string]interface{}))
		}

		fmt.Printf("%d items retrieved!\n", len(items))

		// shuffle by randomly swap items
		for i := range items {
			j := rand.Intn(len(items))
			items[i], items[j] = items[j], items[i]
		}

		for i := range items {
			item := items[i]
			fmt.Println("")
			fmt.Printf("[#%s] \"%s\"\n", item["item_id"], item["resolved_title"])
			fmt.Printf("%s\n", item["resolved_url"])
			fmt.Printf("Added at %s\n", item["time_added"])

			// user action
		nextItem:
			for true {
				fmt.Printf("Action[open(o), archive(a), delete(d), next(n), quit(q)] > ")
				consoleReader := bufio.NewReader(os.Stdin)
				answer, err := consoleReader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				answer = strings.Trim(strings.ToLower(answer), " \n")

				// FIXME: kinda dirty
				itemAction := func(action string) {
					request_url := fmt.Sprintf("%ssend", API_BASE)

					actions, err := json.Marshal([]map[string]string{
						{
							"action":  action,
							"item_id": item["item_id"].(string),
						},
					})
					request_data := url.Values{
						"consumer_key": {cfg.ApiKey},
						"access_token": {cfg.UserToken},
						"actions":      {string(actions)},
					}

					resp, err = http.PostForm(request_url, request_data)
					if err != nil {
						log.Fatal(err)
						os.Exit(1)
					}
					defer resp.Body.Close()

					if resp.StatusCode != 200 {
						log.Fatal(
							fmt.Sprintf("%s item error, StatusCode=[%d], X-Error-Code=[%s]",
								action, resp.StatusCode, resp.Header.Get("X-Error-Code")))
						os.Exit(1)
					}
					fmt.Printf("Item #%s %sd :-)\n", item["item_id"], action)
				}

				switch answer {
				case "o", "open":
					webbrowser.Open(item["resolved_url"].(string))
				case "a", "archive":
					itemAction("archive")
					break nextItem
				case "d", "delete":
					itemAction("delete")
					break nextItem
				case "n", "next":
					break nextItem
				case "q", "quit":
					os.Exit(0)
				}
			}
		}
	}
}
