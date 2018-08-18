# Create own artifact S3 bucket


```bash
aws cloudformation deploy \
   --profile deepimpact-dev-nv \
   --template-file lambda-artifacts.yaml \
   --stack-name LambdaCfArtifacts
```