package main

import (
	"testing"
	"fmt"
	"encoding/json"
)

func TestMap(t *testing.T) {
	/*cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err)
	}*/

	res, _ := json.Marshal(caaRecord("oort.ch.", AwsIssuer))
	fmt.Printf("%s", res)
}

func TestDic(t *testing.T) {
	res := findDictionary(Dictionnary, "t")
	fmt.Printf("%v\n", res)
}