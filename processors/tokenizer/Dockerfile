# Build for local use
# ------------------------
# docker build -f Dockerfile -t tokenizer .

FROM python:3.8-slim

WORKDIR /app

COPY . /app/
COPY ./requirements.txt ./

# Install ps for health check
RUN apt-get update && apt-get install -y procps

RUN pip3 install -r requirements.txt

CMD ["python3", "tokenizer.py"]