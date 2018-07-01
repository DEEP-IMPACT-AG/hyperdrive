# Cog Template

1. If necessary, install the Hyperdrive.
2. Install the `cert-sub.yaml` stack with
    ```bash
    aws cloudformation deploy \
        --stack-name Cert \
        --template-file cert2.yaml
    ```
3. Install the cognito user pool.
    ```bash
    aws cloudformation deploy \
        --stack-name UserPool \
        --tempate-file userpool.yaml
    ```
4. Create the necessary Google identity client.
5. Install the cognito user pool client.
    ```bash
    aws cloudformation deploy \
        --stack-name UserPoolClient \
        --tempate-file userpoolclient.yaml
    ```




  CognitoUserPoolClientSettings:
    Type: Custom::CognitoClientSettings
    Properties:
      ServiceToken:
        Fn::ImportValue:
          !Sub ${HyperdriveCore}-CognitoClientSettings
      UserPoolId: !Ref CognitoUserPool
      UserPoolClientId: !Ref CognitoUserPoolClient