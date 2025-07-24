FROM golang:alpine AS builder

# 設置環境變數
ENV GO111MODULE=on \
GOPROXY=https://proxy.golang.org,direct \
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o partivo_community ./cmd/server/

###################
# multi-stage build
###################
FROM debian:stable-slim

RUN apt update
RUN apt install ca-certificates -y

#COPY ./templates /templates
WORKDIR /app
COPY ./conf /app/conf

COPY --from=builder /build/partivo_community /app/

#RUN set -eux \
#
#    && apt-get update \
#    && apt-get install -y --no-install-recommends netcat \
EXPOSE 8081

ENTRYPOINT ["/app/partivo_community", "-c", "conf/config_docker.yaml"]