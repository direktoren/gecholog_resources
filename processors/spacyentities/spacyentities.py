import spacy
import json
from nats.aio.client import Client as NATS
import asyncio

# pip3 install nats-py spacy
# python3 -m spacy download en_core_web_sm

# Load English tokenizer, POS tagger, parser, NER, and word vectors
nlp = spacy.load("en_core_web_sm")

# HELPERS
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

    print(f"Text to process: '{text_to_process}'", flush=True)

    # Return processed text with spaCy
    return nlp(text_to_process)


# MESSAGE CALLBACK
# will be invoked when a message is received. Sends jsonData to process and publishes the results
async def message_handler(msg):
    subject = msg.subject
    data = msg.data.decode()
    json_data = json.loads(data)

    print(f"Received message with data: '{data}' on subject '{subject}'")

    try:
        processed = await process(json_data)
    except ValueError as e:
        print(e, flush=True)
    
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

    # Connect to NATS server

    # await nc.connect("nats://host.docker.internal:4222") # If nats is on the host machine network
    # await nc.connect("nats://localhost:4222")            # If nats is on the same container / same host
    await nc.connect("nats://gecholog:4222")               # If nats is on the same docker-compose network
    print("Connected to NATS server!", flush=True)

    # Subscribe to subject "coburn.gl.spacyentities"
    await nc.subscribe("coburn.gl.spacyentities", cb=message_handler)

    # Keep the connection alive until a KeyboardInterrupt or another exception
    try:
        while True:
            await asyncio.sleep(1)  # This keeps your run_nats running indefinitely
    except asyncio.CancelledError:
        pass
    finally:
        await nc.close()  # Close NATS connection when exiting the function
        print("NATS connection closed!", flush=True)

# MAIN
def main():
    try:
        asyncio.run(run_nats())  # This will run your asynchronous function and handle the event loop for you
    except KeyboardInterrupt:
        print("Shutting down...", flush=True)

if __name__ == '__main__':
    main()