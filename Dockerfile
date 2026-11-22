# 多架构构建说明：
# 前端与 Go 编译都固定在构建机的原生架构（$BUILDPLATFORM）上执行，
# arm64 通过 aarch64 交叉工具链编译（SQLite 依赖 cgo，不能关）。
# 这样出 arm64 镜像时不需要在 QEMU 里跑 yarn / go，构建时间从十几分钟降到几分钟。
FROM --platform=$BUILDPLATFORM node:22.22.2 as builder

WORKDIR /build

COPY web/package.json .
COPY web/yarn.lock .

RUN yarn config set registry https://registry.npmmirror.com
RUN yarn --frozen-lockfile

COPY ./web .
COPY ./VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_APP_VERSION=$(cat VERSION) npm run build

FROM --platform=$BUILDPLATFORM golang:1.25.9 AS builder2

ARG TARGETARCH

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct

# arm64 需要 aarch64 的 gcc 与目标 libc 头文件做交叉编译；amd64 自带 gcc 直接编。
# 注：libc6-dev-arm64-cross 必须显式装，--no-install-recommends 不会自动带进来，
# 否则 gcc 会去用宿主机的 amd64 头文件，报 bits/libc-header-start.h 找不到。
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        apt-get update \
        && apt-get install -y --no-install-recommends gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
        && rm -rf /var/lib/apt/lists/*; \
    fi

WORKDIR /build
ADD go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=builder /build/build ./web/build
RUN CC=$([ "$TARGETARCH" = "arm64" ] && echo aarch64-linux-gnu-gcc || echo gcc) \
    GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags "-s -w -X 'one-api/common/config.Version=$(cat VERSION)' -extldflags '-static'" -o one-api

FROM alpine

ARG TARGETARCH
# 预置 tiktoken 编码表（cl100k_base 1.6MB、o200k_base 3.6MB）。
# 库默认从 Azure Blob 在线拉取并缓存到临时目录，国内首次启动常要等几分钟才就绪；
# 打进镜像后冷启动可以秒级完成。确实拉不到时只告警，运行时会回退到在线获取。
ARG BUNDLE_TIKTOKEN=true

RUN apk update \
    && apk upgrade \
    && apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates 2>/dev/null || true

RUN mkdir -p /tiktoken && cd /tiktoken && \
    if [ "$BUNDLE_TIKTOKEN" = "true" ]; then \
      for u in \
        https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken \
        https://openaipublic.blob.core.windows.net/encodings/o200k_base.tiktoken; do \
        n=$(echo -n "$u" | sha1sum | cut -d' ' -f1); \
        if wget -q -O "$n.tmp" "$u"; then mv "$n.tmp" "$n"; \
        else rm -f "$n.tmp"; echo "WARN: 编码表 $u 下载失败，运行时将回退到在线获取"; fi; \
      done; \
    fi

ENV TIKTOKEN_CACHE_DIR=/tiktoken

COPY --from=builder2 /build/one-api /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]
