FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder

# THEME selects which frontend to embed: default | classic | hai
ARG THEME=default

WORKDIR /build/web
COPY web/package.json web/bun.lock ./
COPY web/default/package.json ./default/package.json
COPY web/classic/package.json ./classic/package.json
COPY web/hai/package.json ./hai/package.json
RUN bun install --frozen-lockfile
COPY ./web ./
COPY ./VERSION /build/VERSION
# Build only the selected theme, then normalize its output to a fixed path so
# the Go stage can COPY it without ARG interpolation in the source path.
RUN cd ${THEME} && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat /build/VERSION) bun run build \
    && cp -r /build/web/${THEME}/dist /build/web/active-dist

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ARG THEME=default
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git

ADD go.mod go.sum ./
ENV GOPRIVATE=""
ENV GONOPROXY=""
# 替换为大厂的公共代理镜像源
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.baidu.com/,direct
RUN go mod download -x

COPY . .
# Place the built frontend at the path the active build tag embeds.
COPY --from=builder /build/web/active-dist ./web/${THEME}/dist
# default => no build tag; classic/hai => -tags <theme>
RUN GO_TAGS=$([ "$THEME" = "default" ] && echo "" || echo "$THEME") \
    && go build -tags "$GO_TAGS" -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

ARG THEME=default

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources || \
    sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/new-api /
COPY LICENSE NOTICE THIRD-PARTY-LICENSES.md /licenses/
# Pin runtime theme identity to the embedded build.
ENV THEME=${THEME}
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/new-api"]
