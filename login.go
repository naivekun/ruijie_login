package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func encrypt(input string) string {
	e := 65537
	n, _ := new(big.Int).SetString("94dd2a8675fb779e6b9f7103698634cd400f27a154afa67af6166a43fc26417222a79506d34cacc7641946abda1785b7acf9910ad6a0978c91ec84d40b71d2891379af19ffb333e7517e390bd26ac312fe940c340466b4a5d4af1d65c3b5944078f96a1a51a5a53e4bc302818b7c9f63c4a1b07bd7d874cef1c3d4b2f5eb7871", 16)
	m := new(big.Int).SetBytes([]byte(input))
	enc := new(big.Int).Exp(m, big.NewInt(int64(e)), n)
	ret := fmt.Sprintf("%0256s", enc.Text(16))
	return ret
}

func (c *Config) loginLoop(doLoginReqChan chan bool, resultChan chan int) {
	for {
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
	initDataStr := string(initData)

	url_regex := regexp.MustCompile(`top\.self\.location\.href='(https?://.+)/eportal/index.jsp`)
	url_match := url_regex.FindStringSubmatch(initDataStr)
	if len(url_match) != 2 {
		return ERR_LOGIN_FAILED
	}
	ruijie_url := url_match[1]
	formData := strings.TrimPrefix(initDataStr, fmt.Sprintf("<script>top.self.location.href='%s/eportal/index.jsp?", ruijie_url))
	formData = strings.TrimSuffix(formData, "'</script>\r\n")
	if !strings.HasPrefix(formData, "wlanuserip=") {
		return LOGIN_SUCCESS
	}
	cookieResp, err := http.Get(fmt.Sprintf("%s/eportal/nologin.jsp", ruijie_url))
	if err != nil {
		return ERR_LOGIN_FAILED
	}
	cookie := cookieResp.Header.Get("Set-Cookie")
	fmt.Println(cookie)
	mac_reg := regexp.MustCompile(`mac=([0-9a-f]+)&`)
	mac_match := mac_reg.FindStringSubmatch(formData)
	if len(mac_match) != 2 {
		return ERR_LOGIN_FAILED
	}
	mac := mac_match[1]
	form := url.Values{}
	form.Add("userId", c.Accounts[idx].Username)
	form.Add("password", encrypt(fmt.Sprintf("%s>%s", c.Accounts[idx].Password, mac)))
	form.Add("service", "")
	form.Add("queryString", formData)
	form.Add("passwordEncrypt", "true")

	client := http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/eportal/InterFace.do?method=login", ruijie_url), strings.NewReader(form.Encode()))
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
