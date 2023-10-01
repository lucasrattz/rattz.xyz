## --- Builder image --- ##
FROM golang:1.21.1-bookworm AS builder

WORKDIR /rattz.xyz

COPY ./*.go ./

COPY ./go.mod ./

RUN go build -o bin .

## --- Runner image --- ##
FROM debian:bookworm-slim

WORKDIR /rattz.xyz

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY ./static ./static

COPY ./templates ./templates

COPY profile.json ./

COPY --from=builder /rattz.xyz/bin ./

CMD ["./bin"]
