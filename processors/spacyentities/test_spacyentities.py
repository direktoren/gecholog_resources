import pytest
import spacy
from spacyentities import process

# This test DOES NOT MOCK spacy, it uses the real thing.
# Make sure you have followed the spacyentities install instructions before running.
@pytest.mark.asyncio
async def test_process():
    # Define a test case
    json_data = {
        'outbound_payload': {
            'messages': [
                {},
                {'content': 'Who is Mickey Mouse?'}
            ]
        }
    }

    # Call the function with the test case
    doc = await process(json_data)

    # Check that the function returned a spaCy Doc object
    assert isinstance(doc, spacy.tokens.Doc)

    # Check that the Doc object contains the expected text
    assert doc.text == 'Who is Mickey Mouse?'
    assert doc.ents[0].text == 'Mickey Mouse'
    assert doc.ents[0].label_ == 'PERSON'