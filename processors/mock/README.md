# Mock

The `mock` custom processor let's you easily mock and replicate your LLM API responses. It works like this:

-  Make a regular LLM API request to any `gecholog` router, for example `/service/standard/`
-  The `mock` custom processor will record the response payload and response headers
-  Send as many requests you want to `/mock/service/standard/` to get the same response over and over again

The `mock` processor will randomize the response time to resemble an LLM API. You can change this behavior via the `LAMBDA` environment variable.

More information about `gecholog` can be found at [docs.gecholog.ai](https://docs.gecholog.ai/latest).

## Quick Start: Simulate LLM API Responses
### 1. Clone this Git repo

```sh
git clone https://github.com/direktoren/gecholog_resources.git
```

### 2. Set Url of LLM API

```sh
# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

### 3. Start `gecholog` and the `mock` processor

```sh
cd gecholog_resources/processors/mock
docker compose up -d
```

The Docker Compose command starts and configures the LLM Gateway `gecholog` and the processor `mock`. It builds the `mock` container locally. 

### 4. Make the calls

This example will use Azure OpenAI, but you can use any LLM API service.

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment
```

> NOTE: `gecholog` will obfuscate the `Api-Key` HTTP header by default. Read more at [docs.gecholog.ai](https://docs.gecholog.ai/latest).

Send the request to the `/service/standard/` router:
```sh
curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST -d '{
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
}' "http://localhost:5380/service/standard/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15"
```

Expect a response like this. This will be your recorded response for `/mock/service/standard/` requests

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

Now try to make your requests to the mock router `/mock/service/standard/`:

```sh
curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST -d '{
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
}' "http://localhost:5380/mock/service/standard/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15"
```

And every time you should receive your first recorded response:

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

Congratulations, you now have a mock LLM API! 

Take the app down with

```sh
docker compose down -v
```


## Usage

### Record responses

`mock` will store the last response for each router. 

```sh
request1 to /service/standard/ returns answer1
request2 to /service/standard/ returns answer2
request3 to /service/standard/ returns answer3
request4 to /mock/service/standard/ returns answer3
```

`mock` will  separate the responses for each router.
 
```sh
request1 to /service/standard/ returns answer1
request2 to /service/capped/ returns answer2
request3 to /mock/service/standard/ returns answer1
request4 to /mock/service/capped/ returns answer2
```

### Change response time

`mock` will randomize response time using the [Exponential distribution](https://en.wikipedia.org/wiki/Exponential_distribution) with environment variable `LAMBDA`. Set `LAMBDA=0` for disabling the latency which is the default value. The `docker-compose.yml` uses `LAMBDA=0.2` which gives mean value of response time to 500 ms.

### Start `gecholog` and `mock` manually

```sh
# Set the nats token
export NATS_TOKEN=changeme

# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

```sh
# Create a docker network
docker network create gecholog

# Spin up gecholog container
docker run -d -p 5380:5380 -p 4222:4222 \
  --network gecholog --name gecholog \
  --env NATS_TOKEN=$NATS_TOKEN \
  --env AISERVICE_API_BASE=$AISERVICE_API_BASE \
  gecholog/gecholog:latest

# Copy the gl_config to gecholog (if valid it will be applied directly)
# This config tells gecholog when to call mock
docker cp gl_config.json gecholog:/app/conf/gl_config.json

# OPTIONAL: Check that the config file is applied (both statements should produce the same checksum)
shasum -a 256 gl_config.json| cut -d ' ' -f 1
docker exec gecholog ./healthcheck -s gl -p

# Build the processor container
docker build --no-cache -f Dockerfile -t mock .

# Start the processor container
docker run -d \
        --network gecholog --name mock \
        --env NATS_TOKEN=$NATS_TOKEN \
        --env GECHOLOG_HOST=gecholog \
        --env LANBDA=0.2 \
        mock
```

### Response headers

`mock` will store both the response payload and response headers. The headers are coupled to each recorded answer. `gecholog` applies processing rules for headers as documented in [Headers on docs.gecholog.ai](https://docs.gecholog.ai/latest/II.%20Reference/headers/). For example, the `Session-ID` header will be unique for each request to the `/mock/` router

```json
{
    "Session-Id": [
    "TST00001_1709042087441156891_5_0"
  ],
}
```

### Monitor logs in realtime

You can connect to the service bus of `gecholog` container to see the logs from the api calls. 

This command will display the `control` field that `mock` uses to prevent the request from being forwarded to the LLM API

```sh
# Monitor the logger queue & extract the data
nats sub --translate "jq .request.control" -s "$NATS_TOKEN@localhost:4222" "coburn.gl.logger"
```

Sending a first request to `/service/standard/` and three consecutive to `/mock/service/standard/` produces this output

```sh
14:10:56 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
null

[#2] Received on "coburn.gl.logger"
"/service/standard/openai/deployments/gpt4/chat/completions"

[#3] Received on "coburn.gl.logger"
"/service/standard/openai/deployments/gpt4/chat/completions"

[#4] Received on "coburn.gl.logger"
"/service/standard/openai/deployments/gpt4/chat/completions"
```

## Do you want to know more?

Visit [Gecholog.ai](https://www.gecholog.au) and [docs.gecholog.ai](https://docs.gecholog.ai/latests) for more information about the `gecholog` LLM Gateway.