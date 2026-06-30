# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_DATE
ARG GIT_COMMIT

WORKDIR /build

# Cache dependency layer separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and cross-compile for target platform
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-s -w -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${GIT_COMMIT}" \
    -o server \
    ./cmd/server/main.go

# ── Final image ───────────────────────────────────────────────────────────────
# Alpine used for document-service: MongoDB driver benefits from ca-certificates
# and a libc-compatible base for optional CGO BSON acceleration.
FROM alpine:3.20 AS final

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot

COPY --from=builder /build/server /server

EXPOSE 8084

USER nonroot:nonroot

ENTRYPOINT ["/server"]
