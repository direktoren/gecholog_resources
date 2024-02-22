# Charactercount

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
docker build --no-cache -f Dockerfile -t charactercount .

# Start the processor container
docker run -d \
        --network gecholog --name charactercount \
        --env NATS_TOKEN=$NATS_TOKEN \
        --env GECHOLOG_HOST=gecholog \
        charactercount

# Monitor the logger queue & extract the data
nats sub --translate "jq .request.character_count" -s "$NATS_TOKEN@localhost:4222" "coburn.gl.logger"

```

From a different terminal window run

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment

curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST "http://localhost:5380/service/standard/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15" -d'{
  "messages": [
    {
      "role": "system",
      "content": "Assistant is a large language model trained by OpenAI."
    },
    {
      "role": "user",
      "content": "Who were the founders of Microsoft?"
    }
  ],
  "max_tokens": 15
}'
```

and get the response in the first terminal window

```sh
11:10:57 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
179

```