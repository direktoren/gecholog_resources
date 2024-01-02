# Welcome to gecholog.ai resource page

Visit [gecholog.ai](https://www.gecholog.ai) and [docs.gecholog.ai](https://docs.gecholog.ai) for more information on `gecholog`.

## Content

### Processors

| Processor | Description | Language |
|----------|----------|----------|
| charactercount | simple go template processor | golang |
| contentfilter | custom content filter using [detoxify](https://github.com/unitaryai/detoxify) library | python |
| spacyentities | entity tagger using [spaCy](https://spacy.io) library | python |
| tokenizer.py | tokenizer using [GPT2Tokenizer](https://huggingface.co/docs/transformers/model_doc/gpt2#transformers.GPT2Tokenizer) | python |

### Azure

| Resource | Description | Content |
|----------|----------|----------|
| new-gecholog-resource-group | builds a new resource group with gecholog, storage & dashboard | .bicep, .json (arm) |
| gecholog-container-only | deploys latest gecholog container in existing resource group | .bicep |

[![Deploy LLM Gateway gecholog.ai to Azure](http://azuredeploy.net/deploybutton.png)](https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2Fdirektoren%2Fgecholog_resources%2Fmain%2Fazure%2Fnew-gecholog-resource-group%2Fnew-gecholog-resource-group.json)

### AWS

Coming soon...

### Docker

| Resource | Description | Type |
|----------|----------|----------|
| build custom image | Create your docker image | Dockerfile |
| gecholog-ek-dev | Gecholog.ai + Elastic/Kibana bundle | docker-compose.yml |

