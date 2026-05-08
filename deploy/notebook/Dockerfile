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
    CGO_ENABLED=0 go build -o /app/notebook ./cmd/notebook

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/notebook .
USER appuser
EXPOSE 9002
CMD ["/app/notebook"]
