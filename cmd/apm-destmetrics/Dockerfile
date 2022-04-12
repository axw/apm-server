FROM golang:latest AS build
WORKDIR /src/github.com/elastic/apm-server/
COPY go.mod go.sum ./
COPY approvaltest/go.mod approvaltest/go.sum ./approvaltest/
COPY internal/glog/go.mod internal/glog/go.sum ./internal/glog/
COPY internal/otel_collector/go.mod internal/otel_collector/go.sum ./internal/otel_collector/
RUN --mount=type=cache,mode=0755,target=/go/pkg/mod go mod download
COPY . ./
RUN --mount=type=cache,mode=0755,target=/go/pkg/mod --mount=type=cache,mode=0755,target=/root/.cache/go-build \
	go build -o /apm-destmetrics ./cmd/apm-destmetrics

FROM gcr.io/distroless/base-debian10
WORKDIR /
COPY --from=build /apm-destmetrics /
USER nonroot:nonroot
ENTRYPOINT ["/apm-destmetrics"]
