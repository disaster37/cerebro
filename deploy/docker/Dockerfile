# syntax=docker/dockerfile:1

# For reproducible production builds, pin an explicit digest:
# FROM golang:1.26-bookworm@sha256:...
FROM golang:1.26-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/cerebro ./cmd/cerebro

# For reproducible production builds, pin an explicit digest:
# FROM node:22-bookworm@sha256:...
FROM node:22-bookworm AS ui-builder
WORKDIR /src
ENV NUXT_TELEMETRY_DISABLED=1
COPY package.json package-lock.json ./
RUN npm ci --legacy-peer-deps
COPY nuxt.config.ts tsconfig.json ./
COPY app/ app/
RUN npm run build            # emits /src/output/public

# For reproducible production builds, pin an explicit digest:
# FROM alpine:3.24@sha256:...
FROM alpine:3.24
RUN apk add --no-cache ca-certificates \
 && addgroup -g 1000 cerebro \
 && adduser -D -u 1000 -G cerebro -s /sbin/nologin cerebro
WORKDIR /opt/cerebro
COPY --from=go-builder /build/cerebro /cerebro
COPY --from=ui-builder /src/output/public/ output/public/
COPY deploy/docker/conf/application.yaml conf/application.yaml
RUN mkdir -p /data && chown -R cerebro:cerebro /opt/cerebro /data
USER cerebro:cerebro
EXPOSE 9000
# HEALTHCHECK assumes the default basePath: "/". If you run cerebro under a
# custom basePath, update the probe path here (and the k8s/Helm probes).
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO /dev/null http://127.0.0.1:9000/favicon.ico || exit 1
ENTRYPOINT ["/cerebro"]
