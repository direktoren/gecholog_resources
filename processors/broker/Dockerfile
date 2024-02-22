# Build for local use
# ------------------------
# docker build -f Dockerfile -t broker .

FROM golang:1.21-alpine

WORKDIR /app

COPY ./broker.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN go build -o broker

CMD ["./broker"]
