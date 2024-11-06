#can not work on alpine3.14
FROM golang:1.22-bullseye AS builder
WORKDIR /app

# ENV GO111MODULE on
# ENV GOPROXY https://goproxy.cn

# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apt-get update && apt-get install -y make gcc musl-dev git libc-dev build-essential linux-headers-amd64

COPY . .
RUN go mod download
RUN make

FROM golang:1.22.8-alpine3.20

WORKDIR /app
COPY --from=builder /app/*.so /app/
COPY --from=builder /app/config.yml /app/
COPY --from=builder /app/iotex-analyser /app/
ENTRYPOINT ["./iotex-analyser"]
