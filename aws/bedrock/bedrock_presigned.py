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
