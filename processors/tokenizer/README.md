# Tokenizer

`tokenizer` is a processor to apply `GPT2Tokenizer` to the LLM traffic for a generic token count. This could be useful to see how tokens would be calculated differently between LLM services.

- Routers and Search paths for content fields are specified in the `config` data struct. 
- Both request and response traffic are processed
- A `tokenizer` field is added to the log

`tokenizer` illustrates on how to apply one unified token calculation to different LLM models.

## Quick Start: Test the `tokenizer` processor

### 1. Clone this GitHub repo

```sh
git clone https://github.com/direktoren/gecholog_resources.git
```

### 2. Set environment variables

```sh
# Set the nats token (necessary for tokenizer to connect to gecholog)
export NATS_TOKEN=changeme

# Set the gui secret to be able to gecholog web interface
export GUI_SECRET=changeme

# Replace this with the url to your LLM API
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

### 3. Start `gecholog` and the `tokenizer` processor

```sh
cd gecholog_resources/processors/tokenizer
docker compose up -d
```

The Docker Compose command starts and configures the LLM Gateway `gecholog` and the processor `tokenizer`. It builds the `tokenizer` container locally. 

### 4. Monitor the logs

```sh
nats sub --translate "jq .response.tokenizer" -s "$NATS_TOKEN@localhost" "coburn.gl.logger"
```

### 5. Make the calls

This example will use Azure OpenAI, but you can use any LLM API service.

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment
```

Open a new terminal window. Send the request to the `/service/standard/` router:

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

Standard response

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

And see the output from the `tokenizer` processor in the first terminal window

```sh
11:10:57 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
{
  "gpt2": 14
}
```

Take the app down with

```sh
docker compose down -v
```

## Usage

### Start `gecholog` and `tokenizer` manually

```sh
# Set the nats token (necessary for tokenizer to connect to gecholog)
export NATS_TOKEN=changeme

# Set the gui secret to be able to gecholog web interface
export GUI_SECRET=changeme

# Set the target AI service url
export AISERVICE_API_BASE=https://your.openai.azure.com/

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
docker build --no-cache -f Dockerfile -t tokenizer .

# Start the processor container
docker run -d \
        --network gecholog --name tokenizer \
        --env NATS_TOKEN=$NATS_TOKEN \
        --env GECHOLOG_HOST=gecholog \
        tokenizer
```
