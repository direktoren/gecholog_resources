# Build for local use
# ------------------------
# docker build -f Dockerfile -t charactercount .

FROM golang:1.22-alpine as builder

WORKDIR /app

COPY ./charactercount.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN CGO_ENABLED=0 go build -o charactercount

FROM scratch

COPY --from=builder /app/charactercount /charactercount

CMD ["./charactercount"]
