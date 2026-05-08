# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    go mod download
COPY . .
RUN make proto
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /app/runner ./cmd/runner

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/runner .
EXPOSE 9003
CMD ["/app/runner"]
