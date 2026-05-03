FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/storage ./cmd/storage

FROM alpine:3.19
RUN apk --no-cache add ca-certificates su-exec && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/storage .
COPY deploy/storage/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && mkdir -p /app/uploads
EXPOSE 9004
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/storage"]
