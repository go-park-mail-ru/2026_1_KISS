# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server && \
    CGO_ENABLED=0 go build -o /app/migrator ./cmd/migrator

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrator .
COPY migrations/ ./migrations/
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["/app/server"]
