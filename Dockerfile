# Build stage
FROM golang:1.24-alpine AS builder

# Instala certificados CA necessários para HTTPS
RUN apk add --no-cache ca-certificates git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/api

# Production stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /build/main .
COPY --from=builder /build/db/migrations ./db/migrations
RUN chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
CMD ["./main"]