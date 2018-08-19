package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"log"
	"strings"
)

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	ssms := ssm.New(cfg)
	s3s := s3.New(cfg)
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
