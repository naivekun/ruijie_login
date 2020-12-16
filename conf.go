package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	Accounts      []Account `json:"account"`
	TTLInterval   int       `json:"ttlInterval"`
	RetryInterval int       `json:"retryInterval"`
}

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
			Accounts: []Account{{
				Username: "username here",
				Password: "password here",
			}},
			TTLInterval:   30,
			RetryInterval: 30,
		}, "", "\t")
		ioutil.WriteFile(configFileName, configBytes, 0644)
	}
	fmt.Println("default config created")
	os.Exit(0)
	return nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
