# Welcome to gecholog.ai resource page

Visit [gecholog.ai](https://www.gecholog.ai) and [docs.gecholog.ai](https://docs.gecholog.ai) for more information on `gecholog`.

## Content

### Processors

| Processor | Description | Language |
|----------|----------|----------|
| [charactercount.go](processors/charactercount/) | simple go template processor | golang |
| [contentfilter.py](processors/contentfilter/) | custom content filter using [detoxify](https://github.com/unitaryai/detoxify) library | python |
| [spacyentities.py](processors/spacyentities/) | entity tagger using [spaCy](https://spacy.io) library | python |
| [tokenizer.py](processors/tokenizer/) | tokenizer using [GPT2Tokenizer](https://huggingface.co/docs/transformers/model_doc/gpt2#transformers.GPT2Tokenizer) | python |

### Azure



### AWS

Coming soon...

### Docker

| Resource | Description | Type |
|----------|----------|----------|
| [gecholog-ek-dev](docker/gecholog-ek-dev/) | Gecholog.ai + Elastic/Kibana bundle |docker-compose.yml |


- The docker compose file for `gecholog-ek-dev` bundle
- Custom Dockerfile to build `gecholog` container with your own settings
- Processor templates
    - python spacy_entities example
    - golang character_count example