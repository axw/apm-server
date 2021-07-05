# Build and install protoc plugins (go, vtprotobuf)
FROM golang:latest as build
WORKDIR /go/src/genproto
COPY go.mod go.sum ./
RUN go build github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto
RUN go build google.golang.org/protobuf/cmd/protoc-gen-go

# Download, verify, and unpack protoc.
FROM alpine as download
ENV PROTOC_RELEASE=3.17.3
WORKDIR /protoc
COPY protoc.sha256 .
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_RELEASE}/protoc-${PROTOC_RELEASE}-linux-x86_64.zip
RUN sha256sum -c protoc.sha256
RUN unzip -o protoc-${PROTOC_RELEASE}-linux-x86_64.zip

# Create the final image.
FROM gcr.io/distroless/base
COPY --from=build /go/src/genproto/protoc-gen-go-vtproto /usr/bin/protoc-gen-go_vtproto
COPY --from=build /go/src/genproto/protoc-gen-go /usr/bin/
COPY --from=download /protoc/bin/protoc /usr/bin/
COPY --from=download /protoc/include /usr/include
