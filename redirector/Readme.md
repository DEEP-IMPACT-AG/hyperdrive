# Redirector Installation Instruction

```
aws cloudformation package \
    --profile deepimpact-dev \
    --template-file template.yml \
    --s3-bucket lambdacfartifacts-artifactbucket-10yx1c4johw49 \
    --s3-prefix lambda \
    --output-template-file packaged-template.yml
```

```
aws cloudformation deploy \
  --profile deepimpact-dev \
  --template-file packaged-template.yml \
  --capabilities CAPABILITY_IAM \
  --stack-name DeepImpactRedirector \
  --parameter-override RedirectUrl="https://deepimpact.ch"
```