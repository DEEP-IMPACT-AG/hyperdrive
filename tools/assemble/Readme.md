```
aws cloudformation create-change-set \
  --capabilities CAPABILITY_IAM \
  --profile deepimpact-dev-nv \
  --stack-name HyperdriveCoreTest \
  --template-url https://s3.amazonaws.com/us-east-1.test.hyperdrive.sh/cf/hyperdrive/v0.0.0-52-g6fda/hyperdriveCore.yaml \
  --change-set-name stan-1
```