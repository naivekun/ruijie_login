package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

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
