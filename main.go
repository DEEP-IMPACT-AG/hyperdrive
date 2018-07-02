package main

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/gobuffalo/packr"
	"html/template"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func main() {
	box := packr.NewBox("./resources")
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	t, err := template.New("region").Parse(box.String("hello.txt"))
	if err != nil {
		log.Fatal(err.Error())
	}
	t.Execute(os.Stdout, cfg.Region)
	cfs := cloudformation.New(cfg)
	name := "oort.ch."
	err = MakeHostedZone(box, cfs, name, AwsIssuer, CertBotIssuer)
	if err != nil {
		log.Fatal(err)
	}
}