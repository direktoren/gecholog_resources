# Welcome to gecholog.ai resource page

Visit [gecholog.ai](https://www.gecholog.ai) and [docs.gecholog.ai](https://docs.gecholog.ai) for more information on `gecholog`.

## Processors
 
### Resources

- **broker**
    - load balancer and failover processor
    - golang
- **charactercount**
    - simple go template processor
    - golang
- **contentfilter**
    - custom content filter using [detoxify](https://github.com/unitaryai/detoxify) library
    - python
- **mock**
    - Mock the LLM API by replicating your last api call
    - golang
- **regex**
    - versatile regex & json slurper synchronous processor
    - golang
- **spacyentities**
    - entity tagger using [spaCy](https://spacy.io) library
    - python
- **tokenizer**
    - tokenizer using [GPT2Tokenizer](https://huggingface.co/docs/transformers/model_doc/gpt2#transformers.GPT2Tokenizer)
    - python

## Azure

[![Deploy LLM Gateway gecholog.ai to Azure](http://azuredeploy.net/deploybutton.png)](https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2Fdirektoren%2Fgecholog_resources%2Fmain%2Fazure%2Fnew-gecholog-resource-group%2Fnew-gecholog-resource-group.json)

### Resources

- **new-gecholog-resource-group**
    - builds a new resource group with gecholog, storage & dashboard
    - .bicep, .json (arm)
- **gecholog-container-only**
    - deploys latest gecholog container in existing resource group
    - .bicep


## AWS

Coming soon...

## Docker

### Resources

- **gecholog-ek-dev**
    - gecholog.ai container + Elastic/Kibana bundle
    - docker-compose.yml
- **custom docker image**
    - create your docker image
    - Dockerfile


