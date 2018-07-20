# Create own artifact S3 bucket


```bash
aws cloudformation deploy \
   --profile deepimpact-dev \
   --template-file lambda-artifacts.yaml \
   --stack-name LambdaCfArtifacts
```