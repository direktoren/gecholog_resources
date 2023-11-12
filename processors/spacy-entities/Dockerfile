# Build for local use
# ------------------------
# docker build -f docker/container-images/gl-nlp-spacy-entities/Dockerfile -t gl-nlp-spacy-entities .


# Build for Docker Hub
# ------------------------
# docker build -f docker/container-images/gl-nlp-spacy-entities/Dockerfile -t gecholog/gl-nlp-spacy-entities:v0.8.9 . 
#
# Push to Docker Hub
# docker push gecholog/gl-nlp-spacy-entities:v0.8.9
#
# Create a :latest
# docker tag gecholog/gl-nlp-spacy-entities:v0.8.9 gecholog/gl-nlp-spacy-entities:latest
#
# docker push gecholog/gl-nlp-spacy-entities:latest


# Example Dockerfile for Python service
FROM python:3.8-slim

WORKDIR /app

COPY /processorexamples/spacyentities/ /app/
COPY /processorexamples/spacyentities/requirements.txt ./

# Install ps for health check
RUN apt-get update && apt-get install -y procps

RUN pip3 install -r requirements.txt

# Download the spacy model
RUN python -m spacy download en_core_web_sm

CMD ["python3", "spacyentities.py"]