# Contentfilter

The `contentfilter` custom processor illustrates how to create your own filter in front of the LLM API to reject requests according to certain rules or classifications. Routers, patterns and logic for filter can be modified. The default behavior is as follows:

- The field `messages[1].content` of requests to `/service/standard/` is classified using the [detoxify](https://github.com/unitaryai/detoxify) library.
- If any of the non-appropriate parameters of the detoxify classification is > `THRESHOLD` the request will be rejected with a rejection response.
- If not it will be forwarded to the LLM API.

The `THRESHOLD` parameter can be adjusted via an environment variable.

## Disclaimer

It is recommended to run the `contentfilter` locally to explore how it works instead of building a container image. It's a considerable amount of dependencies needed, so for production purpose this would need to be optimized.

When building a `contentfilter` Docker image, the process is slow, and therefor instructions below for docker are recommended only after trying running it locally first. The `contenfilter` container image becomes really big (>8gb).

## Quick Start: Explore a Custom Content Filter

### 1. Clone this GitHub repo

```sh
git clone https://github.com/direktoren/gecholog_resources.git
```

### 2. Set environment variables

```sh
# Set the nats token (necessary for contentfilter to connect to gecholog)
export NATS_TOKEN=changeme

# Replace this with the url to your LLM AP
export AISERVICE_API_BASE=https://your.openai.azure.com/
```

### 3. Download the `contentfilter` dependencies

```sh
cd gecholog_resources/processors/contentfilter
pip install -r requirements.txt
```

Enjoy a warm beverage of your choice whilst waiting.

### 4. Start the `gecholog` LLM Gateway

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
docker cp gl_config.json gecholog:/app/conf/gl_config.json

# OPTIONAL: Check that the config file is applied (both statements should produce the same checksum)
shasum -a 256 gl_config.json| cut -d ' ' -f 1
docker exec gecholog ./healthcheck -s gl -p
```

### 5. Run `contentfilter` 

```sh
python contentfilter.py
````

When ready, the output is

```sh
Connected to NATS server!
```
### 6. Make the calls

This example will use Azure OpenAI, but you can use any LLM API service.

```sh
export AISERVICE_API_KEY=your_api_key
export DEPLOYMENT=your_deployment
```

Send the request to the `/service/standard/` router:

```sh
curl -sS -H "api-key: $AISERVICE_API_KEY" -H "Content-Type: application/json" -X POST -d' {
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
}' "http://localhost:5380/service/standard/openai/deployments/$DEPLOYMENT/chat/completions?api-version=2023-05-15"
```

And receive the response

```json
{
  "is_toxic": true
}
```

### Changing Sensitivity

`contentfilter` uses the `THRESHOLD` environment variable to adjust the sensitivity of the filter.

### `contentfilter` container

The container for `contentfilter` is slow to build and the image is very large. But if you want to try it anyway run


```sh
# Build the processor container (this one is a little heavy...)
docker build --no-cache -f Dockerfile -t contentfilter .

# Start the processor container
docker run -d \
    --network gecholog --name contentfilter \
    --env NATS_TOKEN=$NATS_TOKEN \
    --env GECHOLOG_HOST=gecholog \
    --env THRESHOLD=0.5 \
    contentfilter
```

### Monitor logs in realtime

You can connect to the service bus of `gecholog` container to see the logs from the api calls. 

This command will display the `content_filter` classification field that `contentfilter` creates.

```sh
nats sub --translate "jq .request.content_filter" -s "$NATS_TOKEN@localhost" "coburn.gl.logger"
```

Example

```sh
16:41:43 Subscribing on coburn.gl.logger 
[#1] Received on "coburn.gl.logger"
{
  "toxicity": 0.7701785564422607,
  "severe_toxicity": 0.0008779675699770451,
  "obscene": 0.016887160018086433,
  "threat": 0.001066152355633676,
  "insult": 0.03343857452273369,
  "identity_attack": 0.0014626976335421205
}
```