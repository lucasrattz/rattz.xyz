## --- Builder image --- ##
FROM golang:1.25.0-bookworm AS builder

WORKDIR /rattz.xyz

COPY ./*.go ./

COPY ./go.mod ./

COPY ./static ./static

RUN go build -o bin .

## --- Runner image --- ##
FROM debian:bookworm-slim

WORKDIR /rattz.xyz

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY ./templates ./templates

COPY ./scriptum ./scriptum

COPY ./profile ./profile

RUN mkdir ./gallery

COPY ./gallery/index.go.html ./gallery/index.go.html

COPY profile.json ./

COPY --from=builder /rattz.xyz/bin ./

CMD ["./bin"]
