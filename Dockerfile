# syntax=docker/dockerfile:1
# 国内构建：基础镜像可通过 --build-arg 指向公司 Harbor / 云厂商镜像仓库
ARG GO_IMAGE=golang:1.26-alpine
ARG RUN_IMAGE=alpine:3.20

FROM ${GO_IMAGE} AS builder
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn \
    GOTOOLCHAIN=auto \
    CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/willshark .

FROM ${RUN_IMAGE}
# 时区数据已通过 time/tzdata 编译进二进制，无需 apk 安装（避免依赖外网源）
ENV TZ=Asia/Shanghai
RUN adduser -D -u 10001 app && mkdir -p /app && chown app:app /app
WORKDIR /app
COPY --from=builder /out/willshark ./willshark
COPY --chown=app:app envconfig/*.yaml ./envconfig/
USER app
EXPOSE 8899
ENTRYPOINT ["./willshark"]
# 运行环境，对应 envconfig/<env>_config.yaml
CMD ["dev"]
