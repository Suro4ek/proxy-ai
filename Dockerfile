FROM golang:1.22.4 AS builder

COPY . /src
WORKDIR /src

RUN mkdir -p bin/ && go build -o ./bin/ ./...

FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /src/bin /app

WORKDIR /app
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
EXPOSE 8080
CMD ["./proxy-ai"]