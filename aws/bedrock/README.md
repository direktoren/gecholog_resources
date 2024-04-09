# AWS Bedrock

This page describes how to make requests to AWS Bedrock Runtime models via `gecholog`. AWS API integrations require request signing and it is recommended to use the AWS libraries for the signature process. We will show how to use two different AWS library methods to make requests via `gecholog` to AWS Bedrock Runtime API. 

1. Using `invoke_model` method 
2. Pre-signing the request using `SigV4Auth` 

In these examples we will use `python` library and the `Titan` model, but the method can be used for other SDKs and other models such as `Claude` from Anthropic provided by AWS.

## 1. `invoke_model` method via `gecholog`

Using the `invoke_model` directly has both advantages and disadvantages:

Pro

- Simpler integration code
- Closer coupling with the AWS Bedrock method library
- Always HTTPS

Con 

- Requires empty `/` gecholog router
- Cannot use other routers which can limit Traffic Management capabilities

> We will show the process to use self-signed certificates for HTTPS which is not recommended for production use. Replace the self-signed certificate steps with actions to use your real certificates for production.

### 1.a Start `gecholog` configured for `invoke_model`

We have bundled a set of configuration files for exploring AWS Bedrock with TLS enabled. Pull the files from GitHub and use them to quickly spin up a pre-configured `gecholog` container using Docker.

```sh
git clone https://github.com/direktoren/gecholog_resources.git
cd gecholog_resources/aws/bedrock
```

Run the following command to create self-signed certificates (Not for production use!). Two certification files are created locally: `key.pem` and `cert.pem`. Please store them in the same folder with `gecholog.Dockerfile`.

```sh
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

Use the `gecholog.Dockerfile` to build an image with the necessary config and certificate files to the container image.

```sh
docker build --no-cache -f gecholog.Dockerfile -t gechologaws .
```

Start the gecholog container. Replace `https://bedrock-runtime.your-region.amazonaws.com/` with your own AWS Bedrock Runtime url.

```sh
docker run -d --name gecholog -p 5380:5380 -e AISERVICE_API_BASE=https://bedrock-runtime.your-region.amazonaws.com/ -e NATS2FILE_LOGGER_SUBTOPIC=.logger -v .:/app/log gechologaws
```

> NOTE: Make sure you are using the image `gechologaws` that you created in the build step when starting the `gecholog` container to be used for the `invoke_model` method (TLS enabled)


### 1.b Make a request using `invoke_model`

The following is an example of the python code doing chat completion with Amazon Titan model via `gecholog`. Adapt the code as you need. 

```python
import boto3
import json
from botocore.config import Config
 
proxy_definitions = {
    'https': 'https://localhost:5380', # An empty / router needs to be defined in gl_config.json for gecholog
}
 
my_config = Config(
    proxies=proxy_definitions,
    proxies_config={
        'proxy_use_forwarding_for_https': True
    }
)

boto3.setup_default_session(profile_name='Your-Profile-Name') # Your AWS account profile

# Trust the the self-signed certificate for the proxy for dev purposes
# This is not recommended for production
# You can add certificate information to the Config object to trust the certificate for production
gecholog_client = boto3.client('bedrock-runtime',config=my_config,verify=False)

body = json.dumps({
    "inputText": "Human: explain black holes with 10 words\n\nAssistant:",
    "textGenerationConfig": {
        "maxTokenCount":15,
        "stopSequences":[],
        "temperature":0,
        "topP":0.9,
        },
})

modelId = 'amazon.titan-text-express-v1'
accept = 'application/json'
contentType = 'application/json' # The contentType header is required by gecholog

# Make the request
response = gecholog_client.invoke_model(body=body,modelId=modelId, accept=accept, contentType=contentType)

response_body = json.loads(response.get('body').read())

# output
print(response_body)
```

You should receive the follow json from the code above.

```json
{'inputTextTokenCount': 14, 'results': [{'tokenCount': 15, 'outputText': 'Black holes are regions of spacetime with such strong gravity that nothing can escape', 'completionReason': 'LENGTH'}]}
```


## 2. Pre-signed requests via `gecholog` 

Making presigned requests to AWS Bedrock Runtime via `gecholog` has both advantages and disadvantages:

Pro

- Using AWS native libraries
- More agnostic integration code
- Can use both HTTP and HTTPS
- Take advantage of routing capabilities of `gecholog`

Con 

- Does not use Bedrock methods
- Multi-step process

### 2.a Start `gecholog` 

For pre-signed requests to AWS Bedrock Runtime API the `gecholog/gecholog:latest` image can be used in standard configuration.

```sh
docker run -d --name gecholog -p 5380:5380 -e AISERVICE_API_BASE=https://bedrock-runtime.your-region.amazonaws.com/ -e NATS2FILE_LOGGER_SUBTOPIC=.logger -v .:/app/log gecholog/gecholog:latest
```

It is recommended to use the to update the `tokencounter_config.json` to match the AWS patterns for token reporting:

```json
{
    "usage_fields": [
        {
            "router": "default",
            "patterns": [
                {
                    "field": "prompt_tokens",
                    "pattern": "inbound_payload.inputTextTokenCount"
                },
                {
                    "field": "completion_tokens",
                    "pattern": "inbound_payload.results.0.tokenCount"
                }
            ]
        }
    ]
}
```


### 2.b Make a pre-signed request using

The following is an example of the python code doing chat completion with Amazon Titan model via `gecholog` by pre-signing the request. Adapt the code as you need.

```python
import boto3

from botocore.auth import SigV4Auth
from botocore.awsrequest import AWSRequest
from botocore.session import Session
from botocore.credentials import Credentials

import requests
import json
import os

profile = 'Your-Profile-Name' 

bedrock_url = os.getenv("AISERVICE_API_BASE")
if not bedrock_url:
    bedrock_url = "https://bedrock-runtime.your-region.amazonaws.com/"

api = "model/amazon.titan-text-express-v1/invoke"
gecholog_url = "http://localhost:5380/service/standard/"

# Initialize session using your AWS profile
session = boto3.Session(profile_name=profile)

# Create a request body
body = json.dumps({
    "inputText": "Human: explain black holes with 10 words\n\nAssistant:",
    "textGenerationConfig": {
        "maxTokenCount":15,
        "stopSequences":[],
        "temperature":0,
        "topP":0.9,
        },
})

bedrock_host = bedrock_url.replace("https://", "").rstrip("/")

# Create an AWS request object
request = AWSRequest(
    method="POST",
    url=bedrock_url + api,
    data=body,
    headers={
        "Content-Type": "application/json", # Required by gecholog
        "Host": bedrock_host
    }
)

# Get credentials and region from session
credentials = session.get_credentials()
region = session.region_name

# Sign the request
SigV4Auth(credentials, 'bedrock', region).add_auth(request)

# Send the request using requests library
response = requests.post(gecholog_url+api, headers=dict(request.headers), data=request.data)

# Output the response
print(response.text)
```

You should receive the follow json from the code above.

```json
{'inputTextTokenCount': 14, 'results': [{'tokenCount': 15, 'outputText': 'Black holes are regions of spacetime with such strong gravity that nothing can escape', 'completionReason': 'LENGTH'}]}
```
