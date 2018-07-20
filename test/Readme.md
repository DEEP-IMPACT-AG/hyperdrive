# HyperdriveCore

To install the hyperdrive core, run the following command (or use the
hyperdrive command line tool).

```bash
aws cloudformation create-change-set \
  --profile deepimpact-dev \
  --stack-name HyperdriveCore \
  --template-url https://s3-eu-west-1.amazonaws.com/eu-west-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-18-gcb11/test-hyperdriveCore.yaml \
  --change-set-name creation-hpd-1 \
  --change-set-type CREATE \
  --capabilities CAPABILITY_IAM
  
 aws cloudformation create-change-set \
   --region us-east-1 \
   --profile deepimpact-dev \
   --stack-name HyperdriveCore \
   --template-url https://s3.us-east-1.amazonaws.com/us-east-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-23-gb0d1/test-hyperdriveCore.yaml \
   --change-set-name creation-hpd-1 \
   --change-set-type CREATE \
   --capabilities CAPABILITY_IAM
  
 aws cloudformation execute-change-set \
   --profile deepimpact-dev \
   --change-set-name creation-hpd-1
   
 aws cloudformation execute-change-set \
   --region us-east-1 \
   --profile deepimpact-dev \
   --change-set-name creation-hpd-1
```

To update the hyperdrive:

```bash
aws cloudformation create-change-set \
  --profile deepimpact-dev \
  --stack-name HyperdriveCore \
  --template-url https://s3-eu-west-1.amazonaws.com/eu-west-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-24-g6a3a/test-hyperdriveCore.yaml \
  --change-set-name update-hpd-1 \
  --change-set-type UPDATE \
  --capabilities CAPABILITY_IAM
  
 aws cloudformation execute-change-set \
   --profile deepimpact-dev \
   --change-set-name update-hpd-1
 
 aws cloudformation create-change-set \
   --region us-east-1 \
   --profile deepimpact-dev \
   --stack-name HyperdriveCore \
   --template-url https://s3.us-east-1.amazonaws.com/us-east-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-24-g6a3a/test-hyperdriveCore.yaml \
   --change-set-name update-hpd-1 \
   --change-set-type UPDATE \
   --capabilities CAPABILITY_IAM  
   
 aws cloudformation execute-change-set \
   --region us-east-1 \
   --profile deepimpact-dev \
   --change-set-name
```

To test the core, issue the following commands:

```bash
aws cloudformation deploy \
  --profile deepimpact-dev \
  --stack-name hc-test \
  --template-file cert.yaml \
  --parameter-override FirstImpactIoHostedZone=HostedZone-first-impact-io
```

```bash
aws cloudformation deploy \
  --profile deepimpact-dev \
  --stack-name logs-test \
  --template-file log-group.yaml
```

```bash
aws cloudformation create-change-set \
  --region us-east-1 \
  --profile deepimpact-dev \
  --stack-name CfFolderRewrite \
  --template-url https://s3.us-east-1.amazonaws.com/us-east-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-24-gd139/test-cfRewrite.yaml \
  --change-set-name creation-hpd-1 \
  --change-set-type CREATE \
  --capabilities CAPABILITY_IAM
   
aws cloudformation execute-change-set \
  --region us-east-1 \
  --profile deepimpact-dev \
  --change-set-name
```

static site

```bash
aws cloudformation deploy \
  --region eu-west-1 \
  --profile deepimpact-dev \
  --stack-name static-s3-test \
  --template-file static-site-s3.yaml \
  --parameter-override DomainName=test3-hyperdrive.first-impact.io
```

```bash
aws cloudformation deploy \
  --region eu-west-1 \
  --profile deepimpact-dev \
  --stack-name static-cf-test \
  --template-file static-site-cf.yaml
```