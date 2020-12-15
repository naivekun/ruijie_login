package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	LOGIN_SUCCESS = iota
	ONLINE
	NEED_LOGIN
	ERR_TIMEOUT
	ERR_LOGIN_FAILED
	ERR_WRONG_CREDS
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type Config struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	TTLInterval   int    `json:"ttlInterval"`
	RetryInterval int    `json:"retryInterval"`
}

func loadOrDumpConfig(configFileName string) *Config {
	fmt.Println(configFileName)
	if FileExists(configFileName) {
		fmt.Println("Config OK")
		configBytes, err := ioutil.ReadFile(configFileName)
		if err != nil {
			panic(err)
		}
		ret := &Config{}
		json.Unmarshal(configBytes, ret)
		return ret
	} else {
		fmt.Println("Gen Config")
		configBytes, _ := json.MarshalIndent(&Config{
			Username:      "username here",
			Password:      "password here",
			TTLInterval:   30,
			RetryInterval: 30,
		}, "", "\t")
		ioutil.WriteFile(configFileName, configBytes, 0644)
	}
	fmt.Println("default config created")
	os.Exit(0)
	return nil
}

func (c *Config) connTest() int {
	initReq, err := http.Get("http://www.baidu.com")
	if err != nil {
		return ERR_TIMEOUT
	}
	initData, err := ioutil.ReadAll(initReq.Body)
	initReq.Body.Close()
	if strings.HasPrefix(string(initData), "<script>top.self.location.href=") {
		return NEED_LOGIN
	}
	return ONLINE
}

func (c *Config) connTestLoop(doLoginReqChan chan bool, loginResultChan chan int) {
	for {
		testResult := c.connTest()
		if testResult != ONLINE {
			for {
				log.Println("offline! re-connecting")
				doLoginReqChan <- true
				loginResult := <-loginResultChan
				if loginResult == ERR_TIMEOUT || loginResult == ERR_LOGIN_FAILED {
					time.Sleep(time.Duration(c.RetryInterval) * time.Second)
					break
				}
				if loginResult == LOGIN_SUCCESS {
					break
				}
			}
		} else {
			log.Println("connectivity check success")
			time.Sleep(time.Second * time.Duration(c.TTLInterval))
		}
	}
}

func (c *Config) loginLoop(doLoginReqChan chan bool, resultChan chan int) {
	for {
		<-doLoginReqChan
		loginStatus := c.login()
		switch loginStatus {
		case LOGIN_SUCCESS:
			log.Println("user " + c.Username + " login success")
		case ERR_TIMEOUT:
			log.Println("http request timeout, will retry after " + strconv.Itoa(c.RetryInterval) + " seconds")
		case ERR_LOGIN_FAILED:
			log.Println("http request failed, will retry after " + strconv.Itoa(c.RetryInterval) + " seconds")
		case ERR_WRONG_CREDS:
			log.Println("invalid credentials")
			os.Exit(1)
		}
		resultChan <- loginStatus
	}
}

func (c *Config) login() int {
	initReq, err := http.Get("http://www.baidu.com")
	if err == http.ErrHandlerTimeout {
		return ERR_TIMEOUT
	} else {
		if err != nil {
			return ERR_LOGIN_FAILED
		}
	}
	initData, err := ioutil.ReadAll(initReq.Body)
	initReq.Body.Close()

	formData := strings.TrimPrefix(string(initData), "<script>top.self.location.href='http://192.168.50.3:8080/eportal/index.jsp?")
	formData = strings.TrimSuffix(formData, "'</script>\r\n")
	if !strings.HasPrefix(formData, "wlanuserip=") {
		return LOGIN_SUCCESS
	}

	cookieResp, err := http.Get("http://192.168.50.3:8080/eportal/nologin.jsp")
	if err != nil {
		return ERR_LOGIN_FAILED
	}
	cookie := cookieResp.Header.Get("Set-Cookie")
	fmt.Println(cookie)

	form := url.Values{}
	form.Add("userId", c.Username)
	form.Add("password", c.Password)
	form.Add("service", "")
	form.Add("queryString", formData)
	form.Add("passwordEncrypt", "false")

	client := http.Client{}
	req, _ := http.NewRequest("POST", "http://192.168.50.3:8080/eportal/InterFace.do?method=login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Cookie", cookie)
	resp, err := client.Do(req)
	if err != nil {
		if err == http.ErrHandlerTimeout {
			return ERR_TIMEOUT
		}
		return ERR_LOGIN_FAILED
	}
	respBytes, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(respBytes), `"result":"success"`) {
		return LOGIN_SUCCESS
	}
	return ERR_WRONG_CREDS
}

func main() {
	configFile := flag.String("config", "config.json", "config file")
	flag.Parse()
	config := loadOrDumpConfig(*configFile)
	loginReqChan := make(chan bool, 1)
	loginResultChan := make(chan int, 1)
	go config.loginLoop(loginReqChan, loginResultChan)
	config.connTestLoop(loginReqChan, loginResultChan)
}
