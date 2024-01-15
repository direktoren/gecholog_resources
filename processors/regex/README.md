# Regex

## Usage Markdown

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

# Build the processor container
docker build --no-cache -f "Dockerfile" -t regex .

# Start the processor container
docker run -d \
        --network gecholog --name regex \
        --env NATS_TOKEN="$NATS_TOKEN" \
        --env GECHOLOG_HOST=gecholog \
        regex

# Monitor the logger queue & extract the data
nats sub --translate "jq .response.egress_payload.regex" -s "$NATS_TOKEN@$localhost:4222" "coburn.gl.logger"

```

From a different terminal window run

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment

curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST "http://localhost:5380/markdown/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15" -d'{
  "messages": [
    {
      "role": "system",
      "content": "Assistant is a large language model trained by OpenAI."
    },
    {
      "role": "user",
      "content": "Who are the founders of Microsoft? Please provide answer in 20 words in markdown"
    }
  ],
  "max_tokens": 150
}'|jq
```

With response 

```json
{
  "id": "chatcmpl-8gA57qCBPn7nTOMqhNDIiaFxoCRD7",
  "object": "chat.completion",
  "created": 1705059221,
  "model": "gpt-4",
  "choices": [
    {
      "finish_reason": "stop",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "```markdown\nMicrosoft was founded by Bill Gates and Paul Allen on April 4, 1975.\n```"
      }
    }
  ],
  "usage": {
    "prompt_tokens": 38,
    "completion_tokens": 22,
    "total_tokens": 60
  },
  "system_fingerprint": "fp_6d044fb900",
  "regex": {
    "match": true,
    "sections": [
      {
        "text": "Microsoft was founded by Bill Gates and Paul Allen on April 4, 1975.",
        "object": null
      }
    ]
  }
}
```

and get the response in the first terminal window

```sh
12:33:27 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
{
  "match": true,
  "sections": [
    {
      "text": "Microsoft was founded by Bill Gates and Paul Allen on April 4, 1975.",
      "object": null
    }
  ]
}

```


## Usage with MATCH_JSON

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

# Build the processor container
docker build --no-cache -f "Dockerfile" -t regex .

# Start the processor container
docker run -d \
        --network gecholog --name regex \
        --env NATS_TOKEN="$NATS_TOKEN" \
        --env GECHOLOG_HOST=gecholog \
        --env MATCH_JSON=true \
        regex

# Monitor the logger queue & extract the data
nats sub --translate "jq .response.egress_payload.regex" -s "$NATS_TOKEN@$localhost:4222" "coburn.gl.logger"

```

From a different terminal window run

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment

curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST "http://localhost:5380/json/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15" -d'{
  "messages": [
    {
      "role": "system",
      "content": "Assistant is a large language model trained by OpenAI."
    },
    {
      "role": "user",
      "content": "Who are the founders of Microsoft? Please provide answer in 20 words in json"
    }
  ],
  "max_tokens": 150
}'|jq
```

With response 

```json
{
  "id": "chatcmpl-8gA6hfW1QLmh2MaLTI8J55KraVyBq",
  "object": "chat.completion",
  "created": 1705059319,
  "model": "gpt-4",
  "choices": [
    {
      "finish_reason": "stop",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "```json\n{\n  \"founders_of_microsoft\": [\n    \"Bill Gates\",\n    \"Paul Allen\"\n  ]\n}\n```"
      }
    }
  ],
  "usage": {
    "prompt_tokens": 38,
    "completion_tokens": 27,
    "total_tokens": 65
  },
  "system_fingerprint": "fp_6d044fb900",
  "regex": {
    "match": true,
    "sections": [
      {
        "text": "{\n  \"founders_of_microsoft\": [\n    \"Bill Gates\",\n    \"Paul Allen\"\n  ]\n}",
        "object": {
          "founders_of_microsoft": [
            "Bill Gates",
            "Paul Allen"
          ]
        }
      }
    ]
  }
}
```

and get the response in the first terminal window

```sh
12:35:08 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
{
  "match": true,
  "sections": [
    {
      "text": "{\n  \"founders_of_microsoft\": [\n    \"Bill Gates\",\n    \"Paul Allen\"\n  ]\n}",
      "object": {
        "founders_of_microsoft": [
          "Bill Gates",
          "Paul Allen"
        ]
      }
    }
  ]
}

```