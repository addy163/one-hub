FROM node:18 as builder

WORKDIR /build

# 先复制 package.json 和 yarn.lock 并安装依赖
COPY web/package.json web/yarn.lock ./
RUN yarn --frozen-lockfile

# 再复制剩余的 web 目录内容
COPY web .
COPY VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_APP_VERSION=$(cat VERSION) yarn build

FROM golang:1.22 AS builder2

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux

WORKDIR /build
ADD go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=builder /build/build ./web/build
RUN go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)' -extldflags '-static'" -o one-api

FROM alpine

RUN apk update \
    && apk upgrade \
    && apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates 2>/dev/null || true

COPY --from=builder2 /build/one-api /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]
