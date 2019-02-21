package main

import (
	"fmt"
	"io/ioutil"
)

var mbTest, cfgTest []byte

func init() {
	var err error
	dummy = true
	cfgTest, err = ioutil.ReadFile("test/config")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	mbTest, err = ioutil.ReadFile("test/metrics")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
