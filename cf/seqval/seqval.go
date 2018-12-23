// # Sequence Value
//
// The `seqval` custom resource is used to fetch values from a sequence created by the `seq` custom resource.
// A `seqval` custom resource draw a value from a sequence on creation only.
//
// ## Syntax
//
// To create an `seq` resource, add the following resource to your cloudformation
// template (yaml notation, json is similar)
//
// ```yaml
// MySequenceValue:
//   Type: Custom::SequenceValue
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-SequenceValue
//     Sequence: !Ref Sequence
// ```
//
// ## Properties
//
// `Sequence`
//
// > The name of the sequence to draw a value from
//
// _Type_: String
//
// _Required_: Yes
//
// _Update Requires_: replacement
//
// ## Return Values
//
// `Fn::GetAtt`
//
// The attribute `Value` contains the value that has been drawn from the sequence.
package main

import (
	"context"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

var ssm *awsssm.SSM

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	ssm = awsssm.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The SequenceValueProperties is the main data structure for the resource and
// is defined as a go struct. The struct mirrors the properties as defined above.
// We use the library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type SequenceValueProperties struct {
	Sequence string
}

func sequenceValueProperties(input map[string]interface{}) (SequenceValueProperties, error) {
	var properties SequenceValueProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	if properties.Sequence == "" {
		return properties, errors.New("sequence is required")
	}
	return properties, nil
}

func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := sequenceValueProperties(event.ResourceProperties)
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestCreate:
		return nextValue(ssm, event, properties)
	case cfn.RequestUpdate:
		return nextValue(ssm, event, properties);
	default:
		return "", nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}

func nextValue(ssm *awsssm.SSM, event cfn.Event, properties SequenceValueProperties) (string, map[string]interface{}, error) {
	lockValue := "lock"
	overwrite := true
	pname := properties.Sequence
	for i := 0; i < 3600; i++ {
		param, err := ssm.GetParameterRequest(&awsssm.GetParameterInput{
			Name: &pname,
		}).Send()
		if err != nil {
			return event.PhysicalResourceID, nil, errors.Wrapf(err, "unable to get the parameter %s", pname)
		}
		valueText := *param.Parameter.Value;
		if valueText != "lock" {
			lockedParam, err := ssm.PutParameterRequest(&awsssm.PutParameterInput{
				Name: &pname,
				Value: &lockValue,
				Type: awsssm.ParameterTypeString,
				Overwrite: &overwrite,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "unable to lock the parameter %s", pname)
			}
			if *lockedParam.Version == *param.Parameter.Version + 1 {
				value, err := strconv.ParseUint(valueText, 10, 64)
				if err != nil {
					return event.PhysicalResourceID, nil, errors.Wrapf(err, "value must be a uint64: %s", valueText)
				}
				nextValue := value + 1
				nextValueText := strconv.FormatUint(nextValue, 10);
				_, err = ssm.PutParameterRequest(&awsssm.PutParameterInput{
					Name: &pname,
					Value: &nextValueText,
					Type: awsssm.ParameterTypeString,
					Overwrite: &overwrite,
				}).Send()
				if err != nil {
					return event.PhysicalResourceID, nil, errors.Wrapf(err, "unable to set the parameter %s with the next value", pname)
				}
				physicalId := event.LogicalResourceID + ":" + properties.Sequence
				data := make(map[string]interface{}, 1)
				data["Value"] = valueText
				return physicalId, data, nil
			}
		}
		time.Sleep(time.Second)
	}
	return event.PhysicalResourceID, nil, errors.Errorf("unable to draw a value from the parameter: %s", pname)
}
