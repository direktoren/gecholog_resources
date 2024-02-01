# Build for local use
# ------------------------
# docker build -f Dockerfile -t regex .

FROM golang:1.21-alpine

WORKDIR /app

COPY ./regex.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN go build -o regex

CMD ["./regex"]
