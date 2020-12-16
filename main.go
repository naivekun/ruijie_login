package main

import (
	"flag"
)

const (
	LOGIN_SUCCESS = iota
	ONLINE
	NEED_LOGIN
	ERR_TIMEOUT
	ERR_LOGIN_FAILED
	ERR_WRONG_CREDS
)

func main() {
	configFile := flag.String("config", "config.json", "config file")
	flag.Parse()
	config := loadOrDumpConfig(*configFile)
	loginReqChan := make(chan bool, 1)
	loginResultChan := make(chan int, 1)
	go config.loginLoop(loginReqChan, loginResultChan)
	config.connTestLoop(loginReqChan, loginResultChan)
}
