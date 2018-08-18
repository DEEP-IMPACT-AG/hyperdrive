package dnscert

import (
	"bytes"
	common "github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/test/cf"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/pkg/errors"
	"log"
	"text/template"
)

type AcmTestProperties struct {
	DomainName, San, RegionProperty, TagValue string
}

func TestAcmCertificate() error {
	// 1. we prepare some data for the test, mainly loading a template of a cloudformation template, to create/update
	//    and delete the test stack.
	log.Println("Testing the ACM certificates")
	acmTemplate, err := template.New("test_acm.yaml").ParseFiles("test/dnscert/test_acm.yaml")
	if err != nil {
		return errors.Wrap(err, "could not parse template test_acm.yaml")
	}
	acmStackName := "TestAcm"
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithRegion("eu-west-1"),
	)
	if err != nil {
		return errors.Wrap(err, "could not get aws config with region eu-west-1")
	}
	cfs := cloudformation.New(cfg)
	// 2. we create a stack with a simple certificate
	log.Println(".. Installing a certificate in eu-west-1")
	csn := cf.GenCSN()
	buf := bytes.Buffer{}
	acmTemplate.Execute(&buf, AcmTestProperties{
		DomainName: "test.ch",
		San:        "san.test.ch",
		TagValue:   "test",
	})
	templateBody := string(buf.Bytes())
	cs, err := cfs.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		ChangeSetName: &csn,
		ChangeSetType: cloudformation.ChangeSetTypeCreate,
		StackName:     &acmStackName,
		TemplateBody:  &templateBody,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not create stack %s", acmStackName)
	}
	if err = cf.ExecuteChangeset(cfs, cs); err != nil {
		return err
	}
	log.Println(".. Wait for stabilisation")
	if err = cf.WaitForStableStack(cfs, *cs.StackId); err != nil {
		return err
	}
	// 3. we check the 5 outputs + the arn in the eu-west-1 region.
	stack, err := cf.FetchStack(cfs, acmStackName)
	if err != nil {
		return err
	}
	log.Printf(".. Stack ID: %s\n", *stack.StackId)
	outputs := make(map[string]string, 2)
	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}
	if len(outputs) != 5 {
		return errors.New("expecting 5 output values")
	}
	certificateArn := outputs["CertificateArn"]
	if common.ArnRegion(certificateArn) != "eu-west-1" {
		return errors.Errorf("expecting eu-west-1 in arn %s", certificateArn)
	}
	// 4. we change the tags and have the same certificate as before.
	acms := acm.New(cfg)
	tagValue, err := testTagValue(acms, certificateArn)
	if tagValue != "test" {
		return errors.New("tag value not test")
	}
	carn, err := fetchCertificateArn(cfs, acmStackName)
	if err != nil {
		return err
	}
	if certificateArn != carn {
		return errors.Errorf("expecting same arn, old arn %s, current arn %s", certificateArn, carn)
	}
	err = cf.UpdateStack(cfs, acmStackName, acmTemplate,
		AcmTestProperties{
			DomainName: "test.ch",
			San:        "san.test.ch",
			TagValue:   "test2",
		})
	if err != nil {
		log.Printf("Could not change the tags. skipping. %+v\n", err)
	}
	tagValue, err = testTagValue(acms, certificateArn)
	if tagValue != "test2" {
		return errors.New("tag value not test2")
	}
	carn, err = fetchCertificateArn(cfs, acmStackName)
	if err != nil {
		return err
	}
	if certificateArn != carn {
		return errors.Errorf("expecting same arn, old arn %s, current arn %s", certificateArn, carn)
	}
	// 5. we change the domain name a get a new certificate.
	certificateArn, err = changeCertificateExpectingRegion(cfs, acmTemplate, acmStackName, certificateArn,
		"eu-west-1",
		AcmTestProperties{
			DomainName: "test.test.ch",
			San:        "san.test.ch",
			TagValue:   "test",
		})
	if err != nil {
		return err
	}
	// 6. we change the san and get a new certificate
	certificateArn, err = changeCertificateExpectingRegion(cfs, acmTemplate, acmStackName, certificateArn,
		"eu-west-1",
		AcmTestProperties{
			DomainName: "test.test.ch",
			San:        "san2.test.ch",
			TagValue:   "test",
		})
	if err != nil {
		return err
	}
	// 7. we change the region and get a new certificate
	certificateArn, err = changeCertificateExpectingRegion(cfs, acmTemplate, acmStackName, certificateArn,
		"us-east-1",
		AcmTestProperties{
			DomainName:     "test.test.ch",
			San:            "san2.test.ch",
			TagValue:       "test",
			RegionProperty: "Region: \"us-east-1\"",
		})
	if err != nil {
		return err
	}
	// 8. finally, we delete the stack.
	if err = cf.DeleteStack(cfs, *cs.StackId); err != nil {
		return err
	}
	return nil
}

func testTagValue(acms *acm.ACM, certificateArn string) (string, error) {
	tags, err := acms.ListTagsForCertificateRequest(&acm.ListTagsForCertificateInput{
		CertificateArn: &certificateArn,
	}).Send()
	if err != nil {
		return "", errors.Wrapf(err, "could not fetch tags for certificate %s", certificateArn)
	}
	for _, tag := range tags.Tags {
		if *tag.Key == "test" {
			return *tag.Value, nil
		}
	}
	return "", errors.Errorf("tag test not found for certificate %s", certificateArn)
}

func changeCertificateExpectingRegion(
	cfs *cloudformation.CloudFormation,
	acmTemplate *template.Template,
	acmStackName, certificateArn, expectedRegion string,
	acmProperties AcmTestProperties,
) (string, error) {
	err := cf.UpdateStack(cfs, acmStackName, acmTemplate, acmProperties)
	if err != nil {
		return "", err
	}
	stack, err := cf.FetchStack(cfs, acmStackName)
	log.Printf(".. Stack ID: %s\n", *stack.StackId)
	if err != nil {
		return "", err
	}
	newCertificateArn, err := fetchCertificateArn(cfs, acmStackName)
	if err != nil {
		return "", err
	}
	if newCertificateArn == certificateArn {
		return "", errors.Errorf("certificate did not change for stack %s", acmStackName)
	}
	arnRegion := common.ArnRegion(newCertificateArn)
	if arnRegion != expectedRegion {
		return "", errors.Errorf("expecting %s for the certificate %s", expectedRegion, newCertificateArn)
	}
	return newCertificateArn, nil
}

func fetchCertificateArn(cfs *cloudformation.CloudFormation, acmStackName string) (string, error) {
	stack, err := cf.FetchStack(cfs, acmStackName)
	if err != nil {
		return "", err
	}
	log.Printf(".. Stack ID: %s\n", *stack.StackId)
	for _, output := range stack.Outputs {
		if *output.OutputKey == "CertificateArn" {
			return *output.OutputValue, nil
		}
	}
	return "", errors.Errorf("certificate arn not found for stack %s", acmStackName)
}
