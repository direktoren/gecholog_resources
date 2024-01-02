# Build for local use
# ------------------------
# docker build -f docker/container-images/gl-nlp-spacy-entities/Dockerfile -t gl-nlp-spacy-entities .

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