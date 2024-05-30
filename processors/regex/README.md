# Regex

The `regex` custom processor uses regular expression to extract information from the LLM API response. Fields to extract from and regex patterns to use can be customized. The default behavior is as follows:

- regex attempts to extract TEXT within ```` ```markdown TEXT``` ```` from the response field `choices[0].message.content` response for all routers except the `/json/` router
- It adds a new field to the response indicating if match was successful and the extracted TEXT
- For the `/json/` router `regex` extracts JSON and deserializes the extraction.

The environment variable `MATCH_JSON` toggles deserialization.

## Quick Start: Explore Regex Extraction

### 1. Clone this GitHub repo

```sh
git clone https://github.com/direktoren/gecholog_resources.git
```

### 2. Set environment variables

```sh
# Set the nats token (necessary for regex to connect to gecholog)
export NATS_TOKEN=changeme

# Set the gui secret to be able to gecholog web interface
export GUI_SECRET=changeme

# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

### 3. Start `gecholog` and the `regex` processor

```sh
cd gecholog_resources/processors/regex
docker compose up -d
```

The Docker Compose command starts and configures the LLM Gateway `gecholog` and the processor `regex`. It builds the `regex` container locally. 


### 4. Make the calls

These examples will use Azure OpenAI, but you can use any LLM API service.

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment
```

#### Markdown Extraction

Make a request to the `/markdown/` router and ask the LLM API for response in markdown. Test this with `GPT-4` for best results.

```sh
curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST -d' {
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
}' "http://localhost:5380/markdown/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15"
```

Receive a response like this

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

The `regex` processor has added the fields

```json
{
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

#### JSON Extraction

Make a request to the `/json/` router and ask the LLM API for response in JSON. Test this with `GPT-4` for best results.

```sh
curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST -d' {
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
}' "http://localhost:5380/json/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15"
```

Receive a response like this

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

The `regex` processor has added the fields and deserialized the JSON object

```json
{
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

## Usage

### Field Selection

The `regex` processor uses [gjson](https://github.com/tidwall/gjson) syntax to select response fields to extraction. This makes `regex` LLM API agnostic. Default field selection pattern is `choices[0].message.content`.

### Regular Expression

`regex` uses regular expression to extract patterns from the response fields. `regex` is written in go and uses the [Re2 library Syntax](https://github.com/google/re2/wiki/Syntax).

### Start `gecholog` and `regex` manually

```sh
# Set the nats token (necessary for regex to connect to gecholog)
export NATS_TOKEN=changeme

# Set the gui secret to be able to gecholog web interface
export GUI_SECRET=changeme

# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

```sh
# Create a docker network
docker network create gecholog

# Spin up gecholog container
docker run -d -p 5380:5380 -p 4222:4222 -p 8080:8080 \
  --network gecholog --name gecholog \
  --env NATS_TOKEN=$NATS_TOKEN \
  --env GUI_SECRET=$GUI_SECRET \
  --env AISERVICE_API_BASE=$AISERVICE_API_BASE \
  gecholog/gecholog:latest

# Copy the gl_config to gecholog (if valid it will be applied directly)
docker cp gl_config.json gecholog:/app/conf/gl_config.json

# OPTIONAL: Check that the config file is applied (both statements should produce the same checksum)
shasum -a 256 gl_config.json| cut -d ' ' -f 1
docker exec gecholog ./healthcheck -s gl -p

# Build the processor container
docker build --no-cache -f Dockerfile -t regex .

# Start the processor container
docker run -d \
    --network gecholog --name regex \
    --env NATS_TOKEN=$NATS_TOKEN \
    --env GECHOLOG_HOST=gecholog \
    --env MATCH_JSON=true \
    regex
```

### Monitor logs in realtime

You can connect to the service bus of `gecholog` container to see the logs from the api calls. 

This command will display the `regex` response field the processor is adding to each response

```sh
nats sub --translate "jq .response.egress_payload.regex" -s "$NATS_TOKEN@localhost" "coburn.gl.logger"
```

Example 

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
