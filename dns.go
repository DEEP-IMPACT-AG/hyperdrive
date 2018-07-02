package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"strings"
	"fmt"
	"github.com/gobuffalo/packr"
	"encoding/json"
)

func MakeHostedZone(resources packr.Box, cfs *cloudformation.CloudFormation, name string, issuers ...string) error {
	var cft map[string]interface{}
	dhz := resources.Bytes("dns-hosted-zone.json")
	if err := json.Unmarshal(dhz, &cft); err != nil {
		return err
	}
	if len(issuers) > 0 {
		record := caaRecord(issuers...);
		resources := cft["Resources"].(map[string]interface{})
		resources["CaaRootRecord"] = map[string]interface{}{
			"Type": "AWS::Route53::RecordSet",
			"Properties": record,
		}
	}
	return deployCFT(cfs, hostedZoneStackName(name), cft, KeyVal{Key: "HostedZoneName", Val: name})
}

const (
	AwsIssuer = "amazon.com"
	CertBotIssuer = "letsencrypt.org"
)

func caaRecord(issuers ...string) map[string]interface{} {
	records := make([]string, len(issuers))
	for i, el := range issuers {
		records[i] = caaIssuer(el)
	}
	return map[string]interface{} {
		"Name": map[string]interface{}{"Ref": "HostedZoneName"},
		"HostedZoneId": map[string]string{"Ref": "HostedZone"},
		"ResourceRecords": records,
		"TTL": 300,
		"Type": route53.RRTypeCaa,
	}
}

func caaIssuer(issuer string) string {
	return fmt.Sprintf("0 issue \"%s\"", issuer)
}

func MakeHostedZoneDummy(resources packr.Box, r53s *route53.Route53, cfs *cloudformation.CloudFormation, zoneId string) error {
	name, err := zoneName(r53s, zoneId)
	if err != nil {
		return err
	}
	nameServers, err := nameServers(r53s, zoneId, name)
	if err != nil {
		return err
	}
	return makeDummyCFT(resources, cfs, hostedZoneStackName(name), dnsDummyOutputs(zoneId, name, nameServers)...)
}

func zoneName(r53s *route53.Route53, zoneId string) (string, error) {
	request := route53.GetHostedZoneInput{
		Id: &zoneId,
	}
	res, err := r53s.GetHostedZoneRequest(&request).Send()
	if err != nil {
		return "", err
	}
	return *res.HostedZone.Name, nil
}

func nameServers(r53s *route53.Route53, zoneId, name string) (string, error) {
	request := route53.ListResourceRecordSetsInput{
		HostedZoneId:    &zoneId,
		StartRecordName: &name,
		StartRecordType: route53.RRTypeNs,
	}
	res, err := r53s.ListResourceRecordSetsRequest(&request).Send()
	if err != nil {
		return "", err
	}
	records := res.ResourceRecordSets[0].ResourceRecords
	var sb strings.Builder
	for i, el := range records {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(*el.Value)
	}
	return sb.String(), nil
}

func dnsDummyOutputs(zoneId, zoneName, nameServers string) []KeyVal {
	return []KeyVal{
		{Key: "HostedZoneId", Val: zoneId},
		{Key: "HostedZoneName", Val: zoneName},
		{Key: "NameServers", Val: nameServers},
	}
}

func hostedZoneStackName(name string) string {
	var sb strings.Builder
	sb.WriteString("HostedZone")
	for _, el := range strings.Split(name, ".") {
		if len(el) > 0 {
			sb.WriteString("-")
			sb.WriteString(el)
		}
	}
	return sb.String()
}
