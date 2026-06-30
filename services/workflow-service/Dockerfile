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
# distroless/static includes ca-certificates and tzdata; no shell, no package manager.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/server /server

EXPOSE 8083

USER nonroot:nonroot

ENTRYPOINT ["/server"]
