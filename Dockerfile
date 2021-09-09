FROM golang:1.16 AS builder
ENV GO111MODULE=on
ENV GOPROXY https://goproxy.cn,direct
COPY . /root/togettoyou/
WORKDIR /root/togettoyou/
RUN CGO_ENABLED=0 go build -o jaeger-example .

FROM scratch
COPY --from=builder /root/togettoyou/jaeger-example /root/togettoyou/
WORKDIR /root/togettoyou/
EXPOSE 80
ENTRYPOINT ["./jaeger-example"]