FROM golang:1.17 as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go  ./

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build \
    -o /go/bin/main \
    -ldflags '-s -w'

FROM debian:11-slim as runner
RUN apt update && apt install -y ca-certificates 

COPY --from=builder /go/bin/main /app/main

ENTRYPOINT ["/app/main"]
