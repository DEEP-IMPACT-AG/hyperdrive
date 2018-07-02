package main

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/gobuffalo/packr"
	"html/template"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func main() {
	box := packr.NewBox("./resources")
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("???"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	t, err := template.New("region").Parse(box.String("hello.txt"))
	if err != nil {
		log.Fatal(err.Error())
	}
	t.Execute(os.Stdout, cfg.Region)
	ec2s := ec2.New(cfg)
	cfs := cloudformation.New(cfg)
	MakeDefaultVpcCF(ec2s, cfs)
}