package main

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/gobuffalo/packr"
	"html/template"
	"os"
)

func main() {
	box := packr.NewBox("./resources")
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("stan"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	t, err := template.New("region").Parse(box.String("hello.txt"))
	if err != nil {
		log.Fatal(err.Error())
	}
	t.Execute(os.Stdout, cfg.Region)
}