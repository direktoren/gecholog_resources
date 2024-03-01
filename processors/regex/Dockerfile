# Build for local use
# ------------------------
# docker build -f Dockerfile -t regex .

FROM golang:1.22-alpine as builder

WORKDIR /app

COPY ./regex.go /app/
COPY ./go.mod /app/
COPY ./go.sum /app/

RUN CGO_ENABLED=0 go build -o regex

FROM scratch

COPY --from=builder /app/regex /regex

CMD ["./regex"]
