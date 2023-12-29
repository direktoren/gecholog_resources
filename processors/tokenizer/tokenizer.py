import json
from nats.aio.client import Client as NATS
from transformers import GPT2Tokenizer
import asyncio

# CONFIG
config = {
    "verbose": True,
    "nats_url": "localhost:4222",
    "nats_topic": "coburn.gl.tokenizer",
    "model": "gpt2",
    "patterns" : {
         "/gpt4/": {
            "request": ['ingress_payload', 'messages', 1, 'content'],
            "response": ['inbound_payload', 'choices', 0, 'message', 'content']
        },
        "/gpt35turbo/": {
            "request": ['ingress_payload', 'messages', 1, 'content'],
            "response": ['inbound_payload', 'choices', 0, 'message', 'content']
        },
        "/llama2/": {
            "request": ['ingress_payload', 'messages', 1, 'content'],
            "response": ['inbound_payload', 'output']
        }
    }
}


# Initialize the tokenizer
tokenizer = GPT2Tokenizer.from_pretrained(config["model"])

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

    # Tokenize the text
    tokens = tokenizer.tokenize(text_to_process)

    # Count the tokens
    token_count = len(tokens)

    return token_count


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
            "tokenizer": {
                tokenizer.name_or_path: processed
            }
        }
    except ValueError as e:
        print(e, flush=True)
        # response_data remains an empty array in case of failure

    # Publish processed data back to another subject (e.g., "response.subject")
    await msg.respond(json.dumps(response_data).encode())

# MESSAGE LOOP
async def run_nats():
    nc = NATS()

    # Connect to NATS server
    await nc.connect(config["nats_url"])              
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



