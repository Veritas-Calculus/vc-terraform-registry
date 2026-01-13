FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Use Go module proxy for better download reliability
ENV GOPROXY=https://goproxy.io,https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN go mod download || go mod download || go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs libc6-compat

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p /data/providers

EXPOSE 8080

CMD ["./server"]
