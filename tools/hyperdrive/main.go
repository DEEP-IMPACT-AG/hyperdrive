package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"log"
	"strings"
)
const HyperdriveHome = "/Users/stan/engineering/go/src/github.com/DEEP-IMPACT-AG/hyperdrive"

func main() {
	cmd.Execute()
}

func main() {

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	//box := packr.NewBox(HyperdriveHome)
	ssms := ssm.New(cfg)
	s3s := s3.New(cfg)
	//cfs := cloudformation.New(cfg)
	err = checkInstallation(ssms, s3s)
	if err != nil {
		explainInit(err)
	}
}

const HyperdriveSettingPrefix = "/hyperdrive/setting/"

func checkInstallation(ssms *ssm.SSM, s3s *s3.S3) error {
	organizationDomain, err := fetchHyperdriveOrganizationDomain(ssms)
	if err != nil {
		return err
	}
	regions, err := fetchHyperdriveRegions(ssms)
	if err != nil {
		return err
	}
	for _, region := range regions {
		if err := checkArtifactBuckets(s3s, region, organizationDomain); err != nil {
			return err
		}
	}
	return nil
}

func fetchHyperdriveOrganizationDomain(ssms *ssm.SSM) (string, error) {
	return fetchHyperdriveSetting(ssms, "organizationDomain")
}

func fetchHyperdriveRegions(ssms *ssm.SSM) ([]string, error) {
	regionsText, err := fetchHyperdriveSetting(ssms, "regions")
	if err != nil {
		return nil, err
	}
	var regions []string
	err = json.Unmarshal([]byte(regionsText), &regions)
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func fetchHyperdriveSetting(ssms *ssm.SSM, elements ...string) (string, error) {
	path := settingsPath(elements...)
	parameter, err := ssms.GetParameterRequest(&ssm.GetParameterInput{
		Name: &path,
	}).Send()
	if err != nil {
		return "", errors.Wrapf(err, "parameter %s", path)
	}
	return *parameter.Parameter.Value, nil
}

func settingsPath(elements ...string) string {
	return HyperdriveSettingPrefix + strings.Join(elements, "/")
}

func explainInit(err error) {
	log.Println("Run hyperdrive init")
	log.Fatal(err)
}

func checkArtifactBuckets(s3s *s3.S3, region string, organizationDomain string) error {
	bucketName := fmt.Sprintf("%s.artifacts.hyperdrive.%s", region, organizationDomain)
	versionning, err := s3s.GetBucketVersioningRequest(&s3.GetBucketVersioningInput{
		Bucket: &bucketName,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "bucket %s", bucketName)
	}
	if versionning.Status != s3.BucketVersioningStatusEnabled {
		return errors.Errorf("The bucket %s must have versionning enabled", bucketName)
	}
	return nil
}

func initHyperdrive(box *packr.Box, ssms *ssm.SSM, cfs *cloudformation.CloudFormation, organizationDomain string, regions []string) error {
	if err := writeOrganizationNameParameter(ssms, organizationDomain); err != nil {
		return err
	}
	if err := writeRegionsParameter(ssms, regions); err != nil {
		return err
	}
	for _, 	region := range regions {
		if err := ensureArtifactsBucket(box, cfs, organizationDomain, region); err != nil {
			return err
		}
	}
	return nil
}

func writeOrganizationNameParameter(ssms *ssm.SSM, organizationDomain string) error {
	return writeHyperdriveSetting(ssms, organizationDomain, false, "organizationDomain")
}

func writeRegionsParameter(ssms *ssm.SSM, regions []string) error {
	regionsValue, err := json.Marshal(regions)
	if err != nil {
		return errors.Wrapf(err, "regions %v", regions)
	}
	return writeHyperdriveSetting(ssms, string(regionsValue), true,"regions")
}

func writeHyperdriveSetting(ssms *ssm.SSM, value string, override bool, elements ...string) error {
	parameterPath := settingsPath(elements...)
	_, err := ssms.PutParameterRequest(&ssm.PutParameterInput{
		Name: &parameterPath,
		Type: ssm.ParameterTypeString,
		Overwrite: &override,
		Value: &value,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "parameter %s", parameterPath)
	}
	return nil
}

func ensureArtifactsBucket(box *packr.Box, cfs *cloudformation.CloudFormation, organizationDomain, region string) error {
	return nil
}