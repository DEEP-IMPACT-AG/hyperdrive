package loggrp

import (
	"bytes"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/test/cf"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/pkg/errors"
	"log"
	"text/template"
)

type LogGroupTemplateProperties struct {
	LogGroupName, RegionEuWest1Property, RetentionProperty, TagValue string
}

func TestLogGroup() error {
	// 1. we prepare some data for the test, mainly loading a template of a cloudformation template, to create/update
	//    and delete the test stack.
	log.Println("Testing the log groups")
	logGroupTemplate, err := template.New("test_log_group.yaml").ParseFiles("test/loggrp/test_log_group.yaml")
	if err != nil {
		return errors.Wrap(err, "could not parse template test_log_group.yaml")
	}
	logGroupStackName := "TestLogGroup"
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithRegion("eu-west-1"),
	)
	if err != nil {
		return errors.Wrap(err, "could not get aws config with region eu-west-1")
	}
	logGroupName := "/test/loggrp/test"
	cfs := cloudformation.New(cfg)
	// 2. We create the stack.
	log.Println(".. Creating the log group stack")
	csn := cf.GenCSN()
	buf := bytes.Buffer{}
	logGroupTemplate.Execute(&buf, LogGroupTemplateProperties{LogGroupName: logGroupName, TagValue: "hello"})
	templateBody := string(buf.Bytes())
	cs, err := cfs.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		ChangeSetName: &csn,
		ChangeSetType: cloudformation.ChangeSetTypeCreate,
		StackName:     &logGroupStackName,
		TemplateBody:  &templateBody,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not create the change set for the stack %s", logGroupStackName)
	}
	if err = cf.ExecuteChangeset(cfs, cs); err != nil {
		return err
	}
	log.Println(".. Wait for stabilisation")
	if err = cf.WaitForStableStack(cfs, *cs.StackId); err != nil {
		return err
	}
	// 3. Fetching stack information and testing outputs.
	stack, err := cf.FetchStack(cfs, logGroupStackName)
	log.Printf(".. Stack ID: %s\n", *stack.StackId)
	if err != nil {
		return err
	}
	outputs := make(map[string]string, 2)
	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}
	if len(outputs) != 2 {
		log.Println("Error!!: expecting 2 outputs.")
	}
	// 4. Updating the stack with retention 90 days
	log.Println(".. Retention with 90 days")
	err = cf.UpdateStack(cfs, *stack.StackId, logGroupTemplate, LogGroupTemplateProperties{
		LogGroupName:      logGroupName,
		TagValue:          "hello",
		RetentionProperty: "RetentionInDays: \"90\"",
	})
	if err != nil {
		return err
	}
	if err = checkRetentionPolicy(logGroupName, 90); err != nil {
		return errors.Wrap(err, "retention is not 90 days")
	}
	// 5. Updating the stack with no retention (warning, change to 0)
	log.Println(".. no Retention")
	err = cf.UpdateStack(cfs, *stack.StackId, logGroupTemplate, LogGroupTemplateProperties{
		LogGroupName: logGroupName,
		TagValue:     "hello",
	})
	if err != nil {
		return err
	}
	if err = checkRetentionPolicy(logGroupName, -1); err != nil {
		return errors.Wrap(err, "retention should not be defined")
	}
	// 6. Updating the tags
	log.Println(".. checking tags")
	checkTags(logGroupName, "hello")
	log.Println(".. change tags")
	if err = cf.UpdateStack(cfs, *stack.StackId, logGroupTemplate, LogGroupTemplateProperties{
		LogGroupName: logGroupName,
		TagValue:     "goodbye",
	}); err != nil {
		return err
	}
	if err = checkTags(logGroupName, "goodbye"); err != nil {
		return err
	}
	// 7. Change log group name.
	logGroupName2 := "/test/loggrp/test2"
	err = cf.UpdateStack(cfs, *stack.StackId, logGroupTemplate, LogGroupTemplateProperties{
		LogGroupName: logGroupName2,
		TagValue:     "hello",
	})
	stack, err = cf.FetchStack(cfs, logGroupStackName)
	if err != nil {
		return err
	}
	log.Printf(".. Stack ID: %s\n", *stack.StackId)
	outputs = make(map[string]string, 2)
	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}
	if len(outputs) != 2 {
		return errors.New("expecting 2 outputs.")
	}
	// 8. Deleting the stack
	if err = cf.DeleteStack(cfs, *cs.StackId); err != nil {
		return err
	}
	return nil
}

func checkTags(logGroupName string, tagValue string) error {
	log.Println(".. Checking tags")
	for _, region := range []string{"eu-west-1", "us-east-1"} {
		log.Printf(".... region: %s\n", region)
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		logs := cloudwatchlogs.New(cfg)
		lgs, err := logs.ListTagsLogGroupRequest(&cloudwatchlogs.ListTagsLogGroupInput{
			LogGroupName: &logGroupName,
		}).Send()
		if err != nil {
			return errors.Wrapf(err, "could not fetch the tags for log group %s", logGroupName)
		}
		if lgs.Tags == nil || len(lgs.Tags) != 1 {
			return errors.Errorf("Not correct amount of tags: %+v", lgs.Tags)
		}
		if lgs.Tags["test"] != tagValue {
			return errors.Errorf("Not correct amount of tags: %+v", lgs.Tags)
		}
	}
	return nil
}

func checkRetentionPolicy(logGroupName string, days int64) error {
	log.Println(".. Checking retention")
REGION:
	for _, region := range []string{"eu-west-1", "us-east-1"} {
		log.Printf(".... region: %s\n", region)
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		logs := cloudwatchlogs.New(cfg)
		lgs, err := logs.DescribeLogGroupsRequest(&cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: &logGroupName,
		}).Send()
		if err != nil {
			return errors.Wrapf(err, "could not fetch log groups via prefix %s", logGroupName)
		}
		for _, lg := range lgs.LogGroups {
			if *lg.LogGroupName == logGroupName {
				switch days {
				case -1:
					if lg.RetentionInDays != nil {
						return errors.Errorf("retention must be null for log group %s in region %s", logGroupName, region)
					}
				default:
					if lg.RetentionInDays == nil || *lg.RetentionInDays != days {
						return errors.Errorf("retention is not %d days for log group %s in region %s", days, logGroupName, region)
					}
				}
				continue REGION
			}
		}
		return errors.Errorf("No log group found for name %s in region %s", logGroupName, region)
	}
	return nil
}
