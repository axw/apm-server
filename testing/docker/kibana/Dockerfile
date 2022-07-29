# Build the apm-server binary. The golang image version is kept
# up to date with .go-version by Makefile.
FROM golang:1.17.12 as builder
WORKDIR /src
COPY go.mod go.sum /src/
COPY tools /src/tools
COPY internal/approvaltest/go.mod /src/internal/approvaltest/go.mod
COPY internal/glog/go.mod /src/internal/glog/go.mod
COPY internal/otel_collector/go.mod /src/internal/otel_collector/go.mod
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/go/pkg/mod cd tools && go mod download
COPY Makefile *.mk /src
COPY apmpackage /src/apmpackage
COPY internal/version /src/internal/version
COPY .git /src/.git

ENV GOROOT /usr/local/go
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build-package

FROM docker.elastic.co/kibana/kibana:8.4.0-60171339-SNAPSHOT
COPY --from=builder /src/build/packages/apm-*.zip ./x-pack/plugins/fleet/target/bundled_packages/
