# Build for local use
# ------------------------
# docker build -f Dockerfile -t charactercount .

FROM golang:1.21-alpine

WORKDIR /app

COPY ./charactercount.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN go build -o charactercount

CMD ["./charactercount"]
