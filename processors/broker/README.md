# Broker

## Prerequisites

Make sure you have (at least) two deployments

```sh
gpt4
gpt35turbo
```

If you want to use other deployments, update the `gl_config.json` router->outbound->endpoint:

```sh
"endpoint": "openai/deployments/your_first_deployment/chat/completions?api-version=2023-05-15",
"endpoint": "openai/deployments/your_second_deployment/chat/completions?api-version=2023-05-15",
```

## Usage 

```sh
# Set the nats token
export NATS_TOKEN=changeme

# Set the target AI service url
export AISERVICE_API_BASE=https://your.openai.azure.com/

# Create a docker network
docker network create gecholog

# Spin up gecholog container
docker run -d -p 5380:5380 -p 4222:4222 \
  --network gecholog --name gecholog \
  --env NATS_TOKEN=$NATS_TOKEN \
  --env NATS2FILE_LOGGER_SUBTOPIC=.logger \
  --env AISERVICE_API_BASE=$AISERVICE_API_BASE \
  gecholog/gecholog:latest

# Copy the gl_config to gecholog (if valid it will be applied directly)
docker cp gl_config.json gecholog:/app/conf/gl_config.json

# OPTIONAL: Check that the config file is applied (both statements should produce the same checksum)
shasum -a 256 gl_config.json| cut -d ' ' -f 1
docker exec gecholog ./healthcheck -s gl -p

# Build the processor container
docker build --no-cache -f Dockerfile -t broker .

# Start the processor container
docker run -d \
        --network gecholog --name broker \
        --env NATS_TOKEN=$NATS_TOKEN \
        --env GECHOLOG_HOST=gecholog \
        broker

# Monitor the logger queue & extract the data
nats sub --translate "jq .response.gl_path" -s "$NATS_TOKEN@localhost:4222" "coburn.gl.logger"
```

From a different terminal window run as many times as you like

```sh
export AISERVICE_API_KEY=your_api_key

curl -X POST -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -d '{
    "messages": [
      {
        "role": "system",
        "content": "Assistant is a large language model trained by OpenAI."
      },
      {
        "role": "user",
        "content": "Who are the founders of Microsoft?"
      }
    ],
    "max_tokens": 15
  }' "http://localhost:5380/azure/"|jq
```

With response 

```json
{
  "id": "chatcmpl-8nZCiOLutrIDeVT94lyXkYzdKtkDe",
  "object": "chat.completion",
  "created": 1706824088,
  "model": "gpt-35-turbo",
  "choices": [
    {
      "finish_reason": "length",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The founders of Microsoft are Bill Gates and Paul Allen. They founded Microsoft on"
      }
    }
  ],
  "usage": {
    "prompt_tokens": 29,
    "completion_tokens": 15,
    "total_tokens": 44
  }
}
```

and get the responses in the first terminal window

```sh
17:19:28 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
"/azure/dud/"

[#2] Received on "coburn.gl.logger"
"/azure/gpt4/"

[#3] Received on "coburn.gl.logger"
"/azure/gpt4/"

[#4] Received on "coburn.gl.logger"
"/azure/gpt35turbo/"
```

