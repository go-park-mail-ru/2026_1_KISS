FROM golang:1.26-alpine AS builder
RUN apk --no-cache add protobuf make
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make proto
RUN CGO_ENABLED=0 go build -o /app/notebook ./cmd/notebook

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/notebook .
USER appuser
EXPOSE 9002
CMD ["/app/notebook"]
