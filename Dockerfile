FROM golang:1.15-alpine AS builder

WORKDIR /src

COPY . .

RUN go mod download \
    && GOARCH=amd64 GOOS=linux GOBUILD=CGO_ENABLED=0 go build -ldflags '-w -s' -o monitor


FROM alpine:latest

COPY --from=builder /src/monitor /
COPY --from=builder /src/qqwry.dat /

ENTRYPOINT ["/monitor"]