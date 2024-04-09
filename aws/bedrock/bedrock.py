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

boto3.setup_default_session(profile_name='Your-Profile-Name')

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
contentType = 'application/json'

# Make the request
response = gecholog_client.invoke_model(body=body,modelId=modelId, accept=accept, contentType=contentType)

response_body = json.loads(response.get('body').read())

# output
print(response_body)