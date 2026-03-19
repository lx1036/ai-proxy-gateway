


FROM golang:1.24.2 AS builder


ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /app
COPY . .

RUN export CGO_LDFLAGS_ALLOW='-Wl,--unresolved-symbols=ignore-in-object-files' && \
    go build -ldflags="-s -w" -mod=vendor -o envoy-extension-server cmd/envoy-inferencepool-extension-server




FROM ubuntu:24.04
RUN apt-get update && apt-get -y install wget openssl libssl-dev cmake g++

#ENV NVIDIA_VISIBLE_DEVICES=all
#ENV NVIDIA_DRIVER_CAPABILITIES=utility

COPY --from=builder /app/envoy-extension-server  /usr/bin/envoy-extension-server
RUN chmod +x /usr/bin/envoy-extension-server

CMD ["envoy-extension-server"]
