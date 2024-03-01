# Broker

The `broker` custom processor load balances over multiple LLM APIs (routers) and disables resources from the pool of LLM APIs when requests fail. It is easy to change the load balancing and failover logic. The default behavior is as follows:

- Requests to `/azure/` will be forward to either `/azure/gpt35turbo/` or `/azure/gpt4/` or `/azure/dud/`. The order is random.
- If a request is routed to `/azure/dud/` you will receive an error code. From that point `/azure/dud/` is disabled for 10 minutes.

Disabled time can be change using the `DISABLED_TIME` environment variable. The `/azure/dud/` illustrates what happens if one of the LLM APIs fail.

## Prerequisites

Make sure you have (at least) two deployments. These are the defaults

```sh
gpt4
gpt35turbo
```

If you want to use other deployments, update the `gl_config.json` router->outbound->endpoint:

```sh
"path": "/azure/gpt4/",
    "endpoint": "openai/deployments/your_first_deployment/chat/completions?api-version=2023-05-15",
"path": "/azure/gpt35turbo/",
    "endpoint": "openai/deployments/your_second_deployment/chat/completions?api-version=2023-05-15",
```

## Quick Start: Load Balance LLM APIs
### 1. Clone this GitHub repo

```sh
git clone https://github.com/direktoren/gecholog_resources.git
```

### 2. Set environment variables

```sh
# Set the nats token (necessary for broker to connect to gecholog)
export NATS_TOKEN=changeme

# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

### 3. Start `gecholog` and the `broker` processor

```sh
cd gecholog_resources/processors/broker
docker compose up -d
```

The Docker Compose command starts and configures the LLM Gateway `gecholog` and the processor `broker`. It builds the `broker` container locally. 


### 4. Make the calls

This example will use Azure OpenAI, but you can use any LLM API service.

```sh
export AISERVICE_API_KEY=your_api_key
```

Let's send a request to the `/azure/` router. Run as many times as you like

```sh
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
  }' "http://localhost:5380/azure/"
```

If you get routed to `/azure/gpt4/` or `/azure/gpt35turbo/` you would see a response like this

```json
{
  "id": "chatcmpl-8nZCiOLutrIDeVT94lyXkYzdKtkDe",
  "object": "chat.completion",
  "created": 1706824088,
  "model": "gpt-35-turbo",  // OR gpt-4 
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

When you get routed to `/azure/dud/` you would receive and empty response. From that point `/azure/dud/` is disabled for `DISABLED_TIME` and you cannot hit a second time until it's activated again.

Take the app down with

```sh
docker compose down -v
```

## Usage 

### Load Balancing and Disabled Routers

`broker` will randomly select the router to send the traffic to. Example

```sh
request1 to /azure/ randomly selects /azure/gpt4/
request2 to /azure/ randomly selects /azure/gpt4/
request3 to /azure/ randomly selects /azure/gpt35turbo/
request4 to /azure/ randomly selects /azure/dud/
```

`broker` will remove a router from the selection for `DISABLED_TIME` minutes when a request has failed. Example

```sh
request1 to /azure/ randomly selects /azure/gpt4/
request2 to /azure/ randomly selects /azure/dud/ failed. Disabled for DISABLED_TIME minutes
request3 to /azure/ randomly selects /azure/gpt4/
request4 to /azure/ randomly selects /azure/gpt35turbo/
request5 to /azure/ randomly selects /azure/gpt35turbo/
request6 to /azure/ randomly selects /azure/gpt4/
request7 to /azure/ randomly selects /azure/gpt35turbo/
...
request234 to /azure/ randomly selects /azure/gpt4/
# /azure/dud/ enabled again
request235 to /azure/ randomly selects /azure/gpt35turbo/
request236 to /azure/ randomly selects /azure/gpt35turbo/
request237 to /azure/ randomly selects /azure/dud/ failed. Disabled for DISABLED_TIME minutes
...
```

### Disabled time

`broker` will disable a router after a failed request with environment variable `DISABLED_TIME` minutes. Default is `DISABLED_TIME=10`

### Start `gecholog` and `broker` manually

```sh
# Set the nats token (necessary for broker to connect to gecholog)
export NATS_TOKEN=changeme

# Set the target AI service url
export AISERVICE_API_BASE=https://your.openai.azure.com/

# Create a docker network
docker network create gecholog

# Spin up gecholog container
docker run -d -p 5380:5380 -p 4222:4222 \
  --network gecholog --name gecholog \
  --env NATS_TOKEN=$NATS_TOKEN \
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
        --env DISABLED_TIME=10 \
        broker
```

### Monitor logs in realtime

You can connect to the service bus of `gecholog` container to see the logs from the api calls. 

This command will display the router that `broker` has selected

```sh
# Monitor the logger queue & extract the data
nats sub --translate "jq .response.gl_path" -s "$NATS_TOKEN@localhost" "coburn.gl.logger"
```

Example of output

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

