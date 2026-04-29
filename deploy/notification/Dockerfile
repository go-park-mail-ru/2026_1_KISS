FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/notification ./cmd/notification

FROM alpine:3.19
RUN apk --no-cache add ca-certificates su-exec && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/notification .
COPY deploy/notification/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 9006
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/notification"]
