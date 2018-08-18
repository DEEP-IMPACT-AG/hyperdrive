package main

import (
	"testing"
	"fmt"
	"github.com/gobuffalo/packr"
)

func TestMap(t *testing.T) {
	box := packr.NewBox("./resources")
	fmt.Println(sslCertificateTemplate(box, "first-impact.io", "test3"))
}

func TestHostedZoneName(t *testing.T) {
	fmt.Println(hostedZoneStackName("oort.ch."))
}