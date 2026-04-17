FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/runner ./cmd/runner

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/runner .
USER appuser
EXPOSE 9003
CMD ["/app/runner"]
