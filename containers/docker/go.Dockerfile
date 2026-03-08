FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o person-enrichment-service ./cmd/service

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/person-enrichment-service /usr/local/bin/
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/docs/swagger /app/docs/swagger

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/person-enrichment-service"]