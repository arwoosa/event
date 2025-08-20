FROM golang:1.24-alpine AS builder

# 設置環境變數
ENV GO111MODULE=on \
GOPROXY=https://proxy.golang.org,direct \
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64

WORKDIR /build

COPY . .
RUN go mod download

RUN go build -o partivo_event ./cmd/event-server/

###################
# multi-stage build
###################
FROM scratch

#COPY ./templates /templates
WORKDIR /app
COPY ./conf /app/conf

COPY --from=builder /build/partivo_event /app/

#RUN set -eux \
#
#    && apt-get update \
#    && apt-get install -y --no-install-recommends netcat \
EXPOSE 8081

ENTRYPOINT ["/app/partivo_event"]
CMD ["console", "--config", "conf/config_docker.yaml"]