#can not work on alpine3.14
FROM golang:1.22-alpine AS builder
WORKDIR /app

# ENV GO111MODULE on
# ENV GOPROXY https://goproxy.cn

# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache make gcc musl-dev linux-headers git

COPY . .
RUN go mod download
RUN make

FROM golang:1.22-alpine

WORKDIR /app
COPY --from=builder /app/*.so /app/
COPY --from=builder /app/config.yml /app/
COPY --from=builder /app/iotex-analyser /app/
ENTRYPOINT ["./iotex-analyser"]
