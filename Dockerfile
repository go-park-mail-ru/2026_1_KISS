# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app/migrator ./cmd/migrator

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /app/migrator .
COPY migrations/ ./migrations/

USER appuser

CMD ["/app/migrator"]
