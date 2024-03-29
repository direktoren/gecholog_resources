import os
import json
from nats.aio.client import Client as NATS
from detoxify import Detoxify
import asyncio

# python -m pip install detoxify

# CONFIG
config = {
    "verbose": True,
    "nats_topic": "coburn.gl.contentfilter",
    "model": "original",
    "threshold": 0.5,
    "patterns" : {
         "/service/standard/": {
            "request": ['ingress_payload', 'messages', 1, 'content'],
            "response": ['inbound_payload', 'choices', 0, 'message', 'content']
        }
    }
}

# Load the model
model = None

async def load_model():
    global model
    model = Detoxify(config["model"])

async def main():
    await asyncio.gather(load_model())

# Run the main function
asyncio.run(main())

# HELPERS
def verbose_print(debug, message, **kwargs):
    if not debug or (debug and config["verbose"]):
        print(message, **kwargs)

def get_value_from_path(json_data, path):
    current_element = json_data
    for key in path:
        if isinstance(current_element, dict) and key in current_element:
            current_element = current_element[key]
        elif isinstance(current_element, list) and isinstance(key, int) and key < len(current_element):
            current_element = current_element[key]
        else:
            raise ValueError("could not find text in expected fields. Nothing to process")
    return current_element

def extract_request_payload_text(json_data):
    try:
        pattern = config["patterns"][json_data["gl_path"]]["request"]
        return get_value_from_path(json_data, pattern)    
    except (KeyError, IndexError):
        return None

def extract_response_payload_text(json_data):
    try:
        pattern = config["patterns"][json_data["gl_path"]]["response"]
        return get_value_from_path(json_data, pattern)
    except (KeyError, IndexError):
        return None

# PROCESSOR
# It processes the jsonData through a NLP and returns the results
async def process(json_data):
    # Extract text if it exists in the specified hierarchical paths
    text_to_process = None

    # either request_payload or response_payload should be present, never both at once
    try:
        text_to_process = extract_request_payload_text(json_data)
    except:
        try: 
            text_to_process = extract_response_payload_text(json_data)
        except:
            pass
    
    # If we didn't find any text in the expected fields, raise an exception
    if not text_to_process:
        raise ValueError("could not find text in expected fields. Nothing to process")

    verbose_print(True, f"Text to process: '{text_to_process}'", flush=True)

    # Predict toxicity for a text
    results = model.predict(text_to_process)

    # Determine if the text is toxic based on the threshold
    is_toxic = any(score > config["threshold"] for score in results.values())

    serializable_results = {k: float(v) for k, v in results.items()}
    verbose_print(True, f"Results: {serializable_results}", flush=True)
    return { "is_toxic": is_toxic, "results": serializable_results}


# MESSAGE CALLBACK
# will be invoked when a message is received. Sends jsonData to process and publishes the results
async def message_handler(msg):
    subject = msg.subject
    data = msg.data.decode()
    json_data = json.loads(data)

    verbose_print(True, f"Received a message on '{subject}': {data}", flush=True)

    # Initialize response_data to an empty array
    response_data = []

    try:
        processed = await process(json_data)
        # Update response_data only if processing is successful
        response_data = {
            "content_filter": processed["results"],
            "is_toxic": processed["is_toxic"]
        }
        if processed["is_toxic"]:
            response_data["control"] = { "is_toxic": True }
    except ValueError as e:
        verbose_print(False,e, flush=True)
        # response_data remains an empty array in case of failure

    # Publish processed data back to another subject (e.g., "response.subject")
    await msg.respond(json.dumps(response_data).encode())

# MESSAGE LOOP
async def run_nats():
    nc = NATS()
    glhost = os.getenv("GECHOLOG_HOST")
    if not glhost:
        glhost = "localhost"

    nToken = os.getenv("NATS_TOKEN")

    threshold = os.getenv("THRESHOLD")
    if threshold:
        config["threshold"] = float(threshold)
        verbose_print(False, f"Threshold set to {config['threshold']}", flush=True)

    # Connect to NATS server
    await nc.connect("nats://" + glhost + ":4222", token=nToken) 
    verbose_print(False, "Connected to NATS server!", flush=True)

    # Subscribe to subject nats_topic
    await nc.subscribe(config["nats_topic"], cb=message_handler)

    # Keep the connection alive until a KeyboardInterrupt or another exception
    try:
        while True:
            await asyncio.sleep(1)  # This keeps your run_nats running indefinitely
    except asyncio.CancelledError:
        pass
    finally:
        await nc.close()  # Close NATS connection when exiting the function
        verbose_print(False,"NATS connection closed!", flush=True)

# MAIN
def main():
    try:
        asyncio.run(run_nats())  # This will run your asynchronous function and handle the event loop for you
    except KeyboardInterrupt:
        verbose_print(False,"Shutting down...", flush=True)

if __name__ == '__main__':
    main()



