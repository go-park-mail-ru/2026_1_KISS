FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/runner ./cmd/runner

FROM alpine:3.19
RUN apk --no-cache add ca-certificates su-exec && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/runner .
COPY deploy/runner/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 9003
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/runner"]
