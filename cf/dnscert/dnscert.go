// # Dns Certificate
//
// As of 2018-07-10, cloudformation does not support a ACM SSL
// certification with DNS verification, only the old method via email. This
// custom resource lambda function allows the creation of DNS verified ACM
// SSL certificate.
//
// ## Usage
//
// To use the custom resource in your cloudformation template, you must
// first install the hyperdrive core in your account. Alternativaly, you
// can install it manually. We describe the usage with the hyperdive core.
//
// ### Syntax
//
// To create a new ACM certificate, add the following resource to your
// cloudformation template (yaml notation, json is similar)
//
// ```yaml
// MyCertificate:
//   Type: Custom::DnsCertificate
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-DnsCertificate
//     DomainName: <main-domain-name>
//     Region: <region of the certificate>
//     SubjectAlternativeNames:
//     - <alternative names>
//     - ...
//     Tags:
//     - Key: key
//       Value: value
//     - ...
// ```
//
// ### Properties
//
// `ServiceToken`
//
// > The reference to the ARN of this lambda function; imported via the
// > hyperdrive core stack.
// >
// > _Type_: ARN
// >
// > _Required_: Yes
//
// `DomainName`
//
// > The main domain name for this certificate.
// >
// > _Type_: String
// >
// > _Required_: Yes
// >
// > _Update Requires_: Replacement
//
// `Region`
//
// > The region for the certificate. This is mostly useful to create
// > certificates in the us-east-1 region for stacks that are _not_ in the
// > us-east-1 region and that creates cloudfront distributions. If not
// > specified, it is the region of the stack.
// >
// > _Type_: Region (string)
// >
// > _Required_: No
// >
// > _Update Requires_: Replacement
//
// `SubjectAlternativeNames`
//
// > Additional Domain Names for the certificate.
// >
// > _Type_: List of String
// >
// > _Required_: No
// >
// > _Update Requires_: Replacement
//
// `Tags`
//
// > Tags to apply on the certificate.
// >
// > _Type_: List of Tags (a Tag a a map with keys `Key` and `Value`)
// >
// > _Required_: No
// >
// > _Update Requires_: No interruption.
//
// ### Return Values
//
// `Ref`
//
// The `Ref` intrinsic function gives the ARN of the created certificate
//
// `Fn::GetAtt`
//
// For every domain name (given either through the property `DomainName` or
// the property `SubjectAlternativeNames`, the resource generated 2
// attributes for the CNAME record that is used for validation.
//
// 1. `<domain-name>-RecordName` : the name for the CNAME record.
// 2. `<domain-name>-RecordValue`: the value for the CNAME record.
//
// If you use Route53 for DNS, you can use these attributes to generate
// corresponding records in your HostedZone. The hyperdrive can generate
// cloudformation templates for that purpose.
//
// ### Example
//
// The following yaml fragment create a SSL certificate for the domains
// `test.com` and `hello.test.com` in the region us-east-1.
//
// ```yaml
// TestComCertificate:
//   Type: Custom::DnsCertificate
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-DnsCertificate
//     DomainName: test.com
//     Region: us-east-1
//     SubjectAlternativeNames:
//     - hello.test.com
// ```
//
// The created resouce will have a `Ref` of the form
// `arn:aws:acm:us-east-1:xxxxxxxxx:certificate/yyyyyyyyyyyyyyyyyyyyyyyy`
// and 4 additional attributes, namely:
//
// 1. `test.com-RecordName`: the name of the CNAME record for the
//    certificate validation of the domain `test.com`.
// 2. `test.com-RecordValue`: the value for the CNAME record for the
//    validation of the domain `test.com`.
// 3. `hello.test.com-RecordName`: the name of the CNAME record for the
//    certificate validation of the domain `hello.test.com`.
// 4. `hello.test.com-RecordValue`: the value for the CNAME record for the
//    certification validation of the domain `hello.test.com`
//
// ## Implementation
//
// The implemention of the dnscert lambda uses the
// [AWS Lambda Go](https://github.com/aws/aws-lambda-go) library to
// simplify the integration. It is run in the `go1.x` runtime.
package main

import (
	"context"
	"fmt"
	"github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"time"
)

// The lambda is started using the AWS lambda go sdk. The handler function
// does the actual work of creating the certificate. Cloudformation sends
// an event to signify that a resources must be created, updated or
// deleted.
func main() {
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The main data structure for the certificate resource is defined as a go
// struct. The struct mirrors the properties as defined above. We use the
// library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type DnsCertificateProperties struct {
	DomainName              string
	Region                  string
	SubjectAlternativeNames []string
	Tags                    []acm.Tag
}

func dnsCertificateProperties(input map[string]interface{}) (DnsCertificateProperties, error) {
	var properties DnsCertificateProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	return properties, nil
}

// When processing an event, we first decode the resource properties and
// create a acm client client. We have then 3 cases:
//
// 1. Delete: The delete case it self has 2 sub cases: if the physical
//    resource id is a failure id, then this is a NOP, otherwise we delete
//    the certificate.
// 2. Create: In that case, we proceed to create the certificate,
//    add tags if applicable and collect the DNS CNAME records to construct
//    the attributes of the resource.
// 3. Update: If only the tags have changed, we update them; otherwise, the update
//    requires a replacement and the resource is normally created.
func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := dnsCertificateProperties(event.ResourceProperties)
	if err != nil {
		return "", nil, err
	}
	acms, err := acmService(properties)
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			_, err := acms.DeleteCertificateRequest(&acm.DeleteCertificateInput{
				CertificateArn: &event.PhysicalResourceID,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the certificate %s", event.PhysicalResourceID)
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestCreate:
		return createCertificate(acms, event, properties)
	case cfn.RequestUpdate:
		oldProperties, err := dnsCertificateProperties(event.OldResourceProperties)
		if err != nil {
			return event.PhysicalResourceID, nil, err
		}
		if onlyTagsChanged(event, oldProperties, properties) {
			data, err := updateTags(acms, event, properties)
			return event.PhysicalResourceID, data, err
		} else {
			return createCertificate(acms, event, properties)
		}
	default:
		return event.PhysicalResourceID, nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}

// ### Creation
//
// We create the certificate with the certificate transparency logging
// enabled. If applicable, we add the tags to the certificate. Finally, we
// gather the CNAME record to be exported at attributes of the resource.
func createCertificate(acms *acm.ACM, event cfn.Event, properties DnsCertificateProperties) (string, map[string]interface{}, error) {
	// 1. Create the certificate with certificate transparency logging enabled
	res, err := acms.RequestCertificateRequest(&acm.RequestCertificateInput{
		DomainName:       &properties.DomainName,
		ValidationMethod: acm.ValidationMethodDns,
		Options: &acm.CertificateOptions{
			CertificateTransparencyLoggingPreference: acm.CertificateTransparencyLoggingPreferenceEnabled,
		},
		SubjectAlternativeNames: properties.SubjectAlternativeNames,
	}).Send()
	if err != nil {
		return "", nil, errors.Wrap(err, "could not create the certificate")
	}

	// 2. If applicable, create the tags
	if len(properties.Tags) > 0 {
		_, err = acms.AddTagsToCertificateRequest(&acm.AddTagsToCertificateInput{
			CertificateArn: res.CertificateArn,
			Tags:           properties.Tags,
		}).Send()
		if err != nil {
			return *res.CertificateArn, nil, errors.Wrapf(err, "could not add tags to certificate %s", *res.CertificateArn)
		}
	}

	// 3. Fetch the certificate to get the domain validation information.
	data, err := dataForResource(acms, res.CertificateArn, properties)
	if err != nil {
		return *res.CertificateArn, nil, err
	}

	// 4. Construct the response to cloudformation.
	return *res.CertificateArn, data, nil
}

// Fetching for the data for the CNAME records requires a loop and waiting
// since those are created by AWS asynchronously and added to the
// certificate information only when they have been properly created. We
// wait at most 3 minutes with 3 seconds interval.
func dataForResource(acms *acm.ACM, certificateArn *string, properties DnsCertificateProperties) (map[string]interface{}, error) {
OUTER:
	for i := 0; i < 60; i++ {
		cert, err := acms.DescribeCertificateRequest(&acm.DescribeCertificateInput{
			CertificateArn: certificateArn,
		}).Send()
		if err != nil {
			return nil, errors.Wrapf(err, "could not fetch certificate %s", *certificateArn)
		}
		fmt.Printf("Attempt %d: %+v\n", i, cert)
		options := cert.Certificate.DomainValidationOptions
		if options != nil && len(options) == len(properties.SubjectAlternativeNames)+1 {
			data := make(map[string]interface{}, 2*len(options))
			data["Arn"] = *certificateArn
			for _, option := range options {
				if option.ResourceRecord == nil {
					time.Sleep(3 * time.Second)
					continue OUTER
				}
				domainName := *option.DomainName
				data[domainName+"-RecordName"] = *option.ResourceRecord.Name
				data[domainName+"-RecordValue"] = *option.ResourceRecord.Value
			}
			return data, nil
		}
		time.Sleep(time.Second)
	}
	return nil, errors.Errorf("no DNS entries for certificate %s", *certificateArn)
}

// ### Update
//
// As explained above, we update if and only if the tags are the only
// properties to have changed. For this purpose, we check the equality of
// all the other properties. `SubjectAlternativeNames` is considered a set.
//
// Note that we do not test the tags themselves: it is not necessary as
// cloudformation sends an update request only if at least one property has
// changed.
func onlyTagsChanged(event cfn.Event, oldProperties, properties DnsCertificateProperties) bool {
	return properties.DomainName == oldProperties.DomainName &&
		common.IsSameRegion(event, oldProperties.Region, properties.Region) &&
		sameSubjectAlternativeNames(properties.SubjectAlternativeNames, oldProperties.SubjectAlternativeNames)
}

func sameSubjectAlternativeNames(san1, san2 []string) bool {
	if san1 == nil && san2 == nil {
		return true
	}
	if san1 == nil || san2 == nil {
		return false
	}
	if len(san1) != len(san2) {
		return false
	}
	var san1Set = make(map[string]struct{})
	for _, san := range san1 {
		san1Set[san] = struct{}{}
	}
	for _, san := range san2 {
		_, ok := san1Set[san]
		if !ok {
			return false
		}
	}
	return true
}

// Updating is quite straightforward: we delete all the tags before
// recreating them. We must gather the CNAME records to send as attribute
// to the response.
func updateTags(acms *acm.ACM, event cfn.Event, properties DnsCertificateProperties) (map[string]interface{}, error) {
	// 1. we first fetch the tags.
	tags, err := acms.ListTagsForCertificateRequest(&acm.ListTagsForCertificateInput{
		CertificateArn: &event.PhysicalResourceID,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "could not list tags for certificate %s", event.PhysicalResourceID)
	}
	// 2. we remove them all.
	_, err = acms.RemoveTagsFromCertificateRequest(&acm.RemoveTagsFromCertificateInput{
		CertificateArn: &event.PhysicalResourceID,
		Tags:           tags.Tags,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "could not remove tags for certificate %s", event.PhysicalResourceID)
	}
	// 3. we create the new tags.
	_, err = acms.AddTagsToCertificateRequest(&acm.AddTagsToCertificateInput{
		CertificateArn: &event.PhysicalResourceID,
		Tags:           properties.Tags,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "could not add tags for certificate %s", event.PhysicalResourceID)
	}
	// 4. we gather the data.
	data, err := dataForResource(acms, &event.PhysicalResourceID, properties)
	if err != nil {
		return nil, err
	}
	// 5. finally, we send back the response.
	return data, nil
}

// ### SDK client
//
// We use the
// [ACM sdk v2](https://github.com/aws/aws-sdk-go-v2/tree/master/service/acm)
// to create the certificate. The client is created with the default
// credential chain loader, if need be with the supplied region.
func acmService(properties DnsCertificateProperties) (*acm.ACM, error) {
	var cfg aws.Config
	var err error
	if len(properties.Region) > 0 {
		cfg, err = external.LoadDefaultAWSConfig(external.WithRegion(properties.Region))
		if err != nil {
			return nil, errors.Wrapf(err, "could not load config with region %s", properties.Region)
		}
	} else {
		cfg, err = external.LoadDefaultAWSConfig()
		if err != nil {
			return nil, errors.Wrap(err, "could not load default config")
		}
	}
	return acm.New(cfg), nil
}
