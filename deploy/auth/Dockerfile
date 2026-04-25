FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
RUN echo "Current directory: $(pwd)"
RUN ls -la
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/auth ./cmd/auth

FROM alpine:3.19
RUN apk --no-cache add ca-certificates su-exec && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/auth .
# ДИАГНОСТИКА - посмотрим, что есть в контексте сборки
RUN echo "=== Listing current directory ===" && ls -la
RUN echo "=== Looking for deploy/auth ===" && ls -la ./deploy/ 2>/dev/null || echo "No deploy folder"
RUN echo "=== Searching for entrypoint.sh ===" && find . -name "entrypoint.sh" 2>/dev/null || echo "Not found"
RUN echo "Current directory: $(pwd)"
RUN ls -la
COPY ./deploy/auth/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && mkdir -p /app/uploads
EXPOSE 9001
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/auth"]
