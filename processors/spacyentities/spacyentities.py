import os
import spacy
import json
from nats.aio.client import Client as NATS
import asyncio

# python -m pip install spacy
# python -m spacy download en_core_web_sm

# CONFIG
config = {
    "verbose": False,
    "nats_topic": "coburn.gl.spacyentities",
    "model": "en_core_web_sm",
}

# Load English tokenizer, POS tagger, parser, NER, and word vectors
nlp = spacy.load(config["model"])

# HELPERS
def verbose_print(debug, message, **kwargs):
    if not debug or (debug and config["verbose"]):
        print(message, **kwargs)

def extract_outbound_payload(json_data):
    try:
        return json_data['outbound_payload']['messages'][1]['content']
    except (KeyError, IndexError):
        return None

def extract_egress_payload(json_data):
    try:
        return json_data['egress_payload']['choices'][0]['message']['content']
    except (KeyError, IndexError):
        return None

# PROCESSOR
# It processes the jsonData through a NLP and returns the results
async def process(json_data):
    # Extract text if it exists in the specified hierarchical paths
    text_to_process = None

    # either outbound_payload or egress_payload should be present, never both at once
    text_to_process = extract_outbound_payload(json_data)
    if not text_to_process:
        text_to_process = extract_egress_payload(json_data)

    # If we didn't find any text in the expected fields, raise an exception
    if not text_to_process:
        raise ValueError("could not find text in expected fields. Nothing to process")

    verbose_print(True, f"Text to process: '{text_to_process}'", flush=True)


    # Return processed text with spaCy
    return nlp(text_to_process)


# MESSAGE CALLBACK
# will be invoked when a message is received. Sends jsonData to process and publishes the results
async def message_handler(msg):
    subject = msg.subject
    data = msg.data.decode()
    json_data = json.loads(data)

    verbose_print(True, f"Received a message on '{subject}': {data}", flush=True)

    try:
        processed = await process(json_data)
    except ValueError as e:
        verbose_print(False,e, flush=True)
    
    # Prepare response data
    response_data = {
        "spacy_entities": {
            "entities": [{"text": entity.text, "label": entity.label_} for entity in processed.ents]
        }
    }

    # Publish processed data back to another subject (e.g., "response.subject")
    await msg.respond(json.dumps(response_data).encode())

# MESSAGE LOOP
async def run_nats():
    nc = NATS()
    glhost = os.getenv("GECHOLOG_HOST")
    if not glhost:
        glhost = "localhost"

    nToken = os.getenv("NATS_TOKEN")

    # Connect to NATS server
    await nc.connect("nats://" + glhost + ":4222", token=nToken) 
    verbose_print(False, "Connected to NATS server!", flush=True)

    # Subscribe to subject "coburn.gl.spacyentities"
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