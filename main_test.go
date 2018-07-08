package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"log"
	"testing"
)

func TestMap(t *testing.T) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("deepimpact-dev"),
	)
	if err != nil {
		log.Fatal(err)
	}

	r53s := route53.New(cfg)
	name, err := zoneName(r53s, "Z3KQMBKIBWAQXV")
	fmt.Printf("%v\n", name)
}
