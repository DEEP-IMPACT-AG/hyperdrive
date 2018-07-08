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
  
 aws cloudformation execute-change-set \
   --profile deepimpact-dev \
   --change-set-name creation-hpd-1
```

To update the hyperdrive:

```bash
aws cloudformation create-change-set \
  --profile deepimpact-dev \
  --stack-name HyperdriveCore \
  --template-url https://s3-eu-west-1.amazonaws.com/eu-west-1.hyperdrive.sh/cf/hyperdrive/v0.0.0-18-gcb11/test-hyperdriveCore.yaml \
  --change-set-name update-hpd-1 \
  --change-set-type UPDATE \
  --capabilities CAPABILITY_IAM
  
 aws cloudformation execute-change-set \
   --profile deepimpact-dev \
   --change-set-name update-hpd-1
```


To test the core, issue the following command:

```bash
aws cloudformation deploy \
  --profile deepimpact-dev \
  --stack-name hc-test \
  --template-file cert.yaml \
  --parameter-override OortHostedZone=HostedZone-oort-ch
```