package cf

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"text/template"
	"time"
)

func FetchStack(cfs *cloudformation.CloudFormation, stackName string) (*cloudformation.Stack, error) {
	stacks, err := cfs.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	}).Send()
	if err != nil {
		vErr, ok := errors.Cause(err).(awserr.RequestFailure)
		if ok && vErr.StatusCode() == 400 {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "could not fetch the stack %s", stackName)
	}
	return &stacks.Stacks[0], nil
}

func DeleteStack(cfs *cloudformation.CloudFormation, stackId string) error {
	_, err := cfs.DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: &stackId,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not delete the stack %s", stackId)
	}
	log.Println(".. Wait for stabilisation")
	if err = WaitForStableStack(cfs, stackId); err != nil {
		return err
	}
	return nil
}

func UpdateStack(cfs *cloudformation.CloudFormation, stackId string, template *template.Template, properties interface{}) error {
	csn := GenCSN()
	buf := bytes.Buffer{}
	template.Execute(&buf, properties)
	templateBody := string(buf.Bytes())
	cs, err := cfs.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		ChangeSetName: &csn,
		ChangeSetType: cloudformation.ChangeSetTypeUpdate,
		StackName:     &stackId,
		TemplateBody:  &templateBody,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not create the update change set for the stack %s", stackId)
	}
	if err = ExecuteChangeset(cfs, cs); err != nil {
		return err
	}
	log.Println(".. Wait for stabilisation")
	if err = WaitForStableStack(cfs, *cs.StackId); err != nil {
		return err
	}
	return nil
}

func ExecuteChangeset(cfs *cloudformation.CloudFormation, cs *cloudformation.CreateChangeSetOutput) error {
	err := waitForChangeSet(cfs, cs)
	if err != nil {
		return err
	}
	_, err = cfs.ExecuteChangeSetRequest(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.Id,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not execute the change set %s", *cs.Id)
	}
	return nil
}

func waitForChangeSet(cfs *cloudformation.CloudFormation, cs *cloudformation.CreateChangeSetOutput) error {
	for i := 0; i < 100; i++ {
		request := cloudformation.DescribeChangeSetInput{
			ChangeSetName: cs.Id,
		}
		res, err := cfs.DescribeChangeSetRequest(&request).Send()
		if err != nil {
			return errors.Wrapf(err, "could not fetch the change set %s", *cs.Id)
		}
		if res.ExecutionStatus == "AVAILABLE" {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.Errorf("cs %s never ready", *cs.Id)
}

func WaitForStableStack(cfs *cloudformation.CloudFormation, stackName string) error {
	// wait at most 1 hour.
	for i := 0; i < 1200; i++ {
		stack, err := FetchStack(cfs, stackName)
		if err != nil {
			return err
		}
		status := stack.StackStatus
		switch status {
		case
			cloudformation.StackStatusCreateComplete,
			cloudformation.StackStatusDeleteComplete,
			cloudformation.StackStatusUpdateComplete:
			return nil
		case
			cloudformation.StackStatusCreateFailed,
			cloudformation.StackStatusRollbackFailed,
			cloudformation.StackStatusRollbackComplete,
			cloudformation.StackStatusDeleteFailed,
			cloudformation.StackStatusUpdateRollbackFailed,
			cloudformation.StackStatusUpdateRollbackComplete:
			return errors.Errorf("error final status %s for stack %s", status, stackName)
		default:
			time.Sleep(3 * time.Second)
		}
	}
	return errors.Errorf("stack %s never stablized", stackName)
}

func GenCSN() string {
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("hyperdrive-%s", uuid.String())
}
