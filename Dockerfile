FROM golang:1.15 as builder
#author
MAINTAINER 826895143@qq.com
WORKDIR /workspace

COPY go.mod go.mod

RUN  go env -w GO111MODULE=on \
    && go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct

RUN go mod download

COPY main.go main.go
COPY app/ app/
COPY consts/ consts/
COPY core/ core/
COPY di/  di/
COPY envconfig/ envconfig/

EXPOSE 8899

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o will main.go

FROM centos:7.2.1511
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo 'Asia/Shanghai' >/etc/timezone

WORKDIR /
COPY --from=builder /workspace/will .

ENTRYPOINT ["/will"]
CMD ["will"]
