# Build for local use
# ------------------------
# docker build -f Dockerfile -t broker .

FROM golang:1.22-alpine as builder

WORKDIR /app

COPY ./broker.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN CGO_ENABLED=0 go build -o broker

FROM scratch

COPY --from=builder /app/broker /broker

CMD ["./broker"]
