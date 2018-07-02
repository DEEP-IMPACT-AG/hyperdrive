package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"strings"
)

func MakeHostedZone() {

}

func MakeHostedZoneDummy(r53s *route53.Route53, cfs *cloudformation.CloudFormation, zoneId string) error {
	name, err := zoneName(r53s, zoneId)
	if err != nil {
		return err
	}
	nameServers, err := nameServers(r53s, zoneId, name)
	if err != nil {
		return err
	}
	return makeDummyCFT(cfs, hostedZoneStackName(name), dnsDummyOutputs(zoneId, name, nameServers)...)
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

func dnsDummyOutputs(zoneId, zoneName, nameServers string) []dummyOutput {
	return []dummyOutput{
		{Key: "HostedZoneId", Val: zoneId},
		{Key: "HostedZoneName", Val: zoneName},
		{Key: "NameServers", Val: nameServers},
	}
}

func hostedZoneStackName(name string) string {
	var sb strings.Builder
	sb.WriteString("HZ")
	for _, el := range strings.Split(name, ".") {
		if len(el) > 0 {
			sb.WriteString("-")
			sb.WriteString(el)
		}
	}
	return sb.String()
}
