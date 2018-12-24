package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"log"
	"strings"
)

// CognitoEventUserPoolsPreSignupRequest contains the request portion of a PreAuth event
type CognitoEventUserPoolsPreAuth struct {
	events.CognitoEventUserPoolsHeader
	Request  events.CognitoEventUserPoolsPreSignupRequest `json:"request"`
	Response map[string]interface{}
}

var zero = CognitoEventUserPoolsPreAuth{}

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatalf("could not get aws config: %+v\n", err)
	}
	ssms := ssm.New(cfg)
	lambda.Start(processEvent(ssms))
}

type Settings struct {
	All     bool     `json:"all"`
	Domains []string `json:"domains"`
	Emails  []string `json:"emails"`
}

func processEvent(ssms *ssm.SSM) func(CognitoEventUserPoolsPreAuth) (CognitoEventUserPoolsPreAuth, error) {
	return func(event CognitoEventUserPoolsPreAuth) (CognitoEventUserPoolsPreAuth, error) {
		fmt.Printf("%+v\n", event)
		userPoolId := event.UserPoolID
		clientId := event.CallerContext.ClientID
		email := event.Request.UserAttributes["email"]
		splitted := strings.Split(email, "@")
		if len(splitted) != 2 {
			return zero, errors.Errorf("invalid email: %s", email)
		}
		domain := splitted[1]
		parameterName := "/hyperdrive/cog_cond_pre_auth/" + userPoolId + "/" + clientId
		parameter, err := ssms.GetParameterRequest(&ssm.GetParameterInput{
			Name: &parameterName,
		}).Send()
		if err != nil {
			return zero, errors.Wrap(err, "DDB fetching error")
		}
		if parameter.Parameter == nil || parameter.Parameter.Value == nil {
			return zero, errors.Errorf("no configuration for the client %s of user pool %s", clientId, userPoolId)
		}
		var settings Settings
		err = json.Unmarshal([]byte(*parameter.Parameter.Value), &settings)
		if err != nil {
			return zero, errors.Wrapf(err, "invalid settings for the client %s of user pool %s", clientId, userPoolId)
		}
		if !(settings.All || in(settings.Domains, domain) || in(settings.Emails, email)) {
			return zero, errors.New("not authorized.")
		}
		return event, nil
	}
}

func in(strings []string, val string) bool {
	for _, s := range strings {
		if val == s {
			return true
		}
	}
	return false
}
