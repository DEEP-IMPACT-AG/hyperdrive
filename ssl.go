package main

import (
	"github.com/gobuffalo/packr"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"strings"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func MakeSSLCertificate(resources packr.Box, cfs *cloudformation.CloudFormation, rootDomain, subDomain string) error {
	template := sslCertificateTemplate(resources, rootDomain, subDomain)
	return deployCFT(cfs, sslCertificateStackName(rootDomain, subDomain), template)
}

func sslCertificateStackName(rootDomain, subDomain string) string {
	var sb strings.Builder
	sb.WriteString("SSLCert-")
	sb.WriteString(subDomain)
	for _, el := range strings.Split(rootDomain, ".") {
		if len(el) > 0 {
			sb.WriteString("-")
			sb.WriteString(el)
		}
	}
	return sb.String()
}


func sslCertificateTemplate(resources packr.Box, rootDomain, subDomain string) map[string]interface{} {
	hostedZoneStackName := hostedZoneStackName(rootDomain)
	ht := hyperdriveTemplate(resources)
	res := make(map[string]interface{}, 2)
	domain := subDomain + "." + rootDomain
	res["Certificate"] = certificateResource(resources, domain)
	res["RecordSetGroup"] = recordSetGroup(resources, hostedZoneStackName, domain)
	ht["Resources"] = res
	ht["Outputs"] = resource(resources, "dns-certificate-outputs.json")
	return ht
}

func certificateResource(resources packr.Box, domain string) interface{} {
	res := resource(resources, "dns-certificate-resource.json")
	p := res["Properties"].(map[string]interface{})
	p["DomainName"] = domain
	p["Region"] = "us-east-1"
	return res
}

func recordSetGroup(resources packr.Box, hostedZoneStackName, domain string) interface{} {
	res := resource(resources, "record-set-group-resource.json")
	p := res["Properties"].(map[string]interface{})
	p["HostedZoneId"] = map[string]interface{}{"Fn::ImportValue": fmt.Sprintf("%s-HostedZoneId", hostedZoneStackName)}
	p["RecordSets"] = recordSets(resources, domain)
	return res
}

func recordSets(resources packr.Box, domain string) interface{} {
	set := make([]interface{}, 2)
	set[0] = cnameRecord(domain)
	set[1] = caaRecord2(resources, domain)
	return set
}

func cnameRecord(domain string) interface{} {
	rec := make(map[string]interface{})
	rec["Name"] = getAtt("Certificate", domain + "-RecordName")
	rec["ResourceRecords"] = []interface{}{getAtt("Certificate", domain + "-RecordValue")}
	rec["TTL"] = "300"
	rec["Type"] = route53.RRTypeCname
	return rec
}

func getAtt(items ...string) interface{} {
	if len(items) != 2 {
		panic(items)
	}
	return map[string]interface{} {
		"Fn::GetAtt": items,
	}
}

func caaRecord2(resources packr.Box, domain string) interface{} {
	rec := resource(resources, "record-set-group-caa-record.json")
	rec["Name"] = domain
	return rec
}