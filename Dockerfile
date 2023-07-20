ARG GO_VERSION=${GO_VERSION:-1.19}

FROM --platform=${BUILDPLATFORM:-linux/amd64}  golang:${GO_VERSION}-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /src
RUN cat /etc/passwd | grep nobody > /etc/passwd.nobody
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download

# Build the binary.
RUN --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -tags=nomsgpack -o "/healthcheck-pinger" .


# build a small image
FROM --platform=${BUILDPLATFORM:-linux/amd64}  alpine

COPY --from=builder "/healthcheck-pinger" "/healthcheck-pinger"
WORKDIR /

ENTRYPOINT ["/healthcheck-pinger"]
