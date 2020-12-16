package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func (c *Config) loginLoop(doLoginReqChan chan bool, resultChan chan int) {
	for i := range c.Accounts {
		<-doLoginReqChan
		loginStatus := c.login(i)
		switch loginStatus {
		case LOGIN_SUCCESS:
			log.Println("user " + c.Accounts[i].Username + " login success")
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

func (c *Config) login(idx int) int {
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
	form.Add("userId", c.Accounts[idx].Username)
	form.Add("password", c.Accounts[idx].Password)
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
