# Build for local use
# ------------------------
# docker build -f Dockerfile -t mock .

FROM golang:1.22-alpine as builder

WORKDIR /app

COPY ./mock.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN CGO_ENABLED=0 go build -o mock

FROM scratch

COPY --from=builder /app/mock /mock

CMD ["./mock"]
