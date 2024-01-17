# Contentfilter

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

# Get the container id
docker ps

# Copy the gl_config file & restart gecholog
docker cp "gl_config.json" your_container_id:/app/conf/gl_config.json
docker restart your_container_id

# Build the processor container (this one is a little heavy...)
docker build --no-cache -f "Dockerfile" -t contentfilter .

# Start the processor container
docker run -d \
        --network gecholog --name contentfilter \
        --env NATS_TOKEN="$NATS_TOKEN" \
        --env GECHOLOG_HOST=gecholog \
        contentfilter

# Monitor the logger queue & extract the contentfilter data
nats sub --translate "jq .response.egress_payload" -s "$NATS_TOKEN@localhost:4222" "coburn.gl.logger"

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
      "content": "Im sick of this nonsense!"
    }
  ],
  "max_tokens": 15
}'
```

and get the response in the first terminal window

```sh
11:34:56 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
{
  "is_toxic": true
}

```