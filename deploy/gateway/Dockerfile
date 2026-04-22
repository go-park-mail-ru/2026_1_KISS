FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/gateway ./cmd/gateway

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/gateway .
RUN mkdir -p /app/uploads && chown appuser:appuser /app/uploads
USER appuser
EXPOSE 8080
CMD ["/app/gateway"]
