# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/node-agent .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates docker-cli

# Install sing-box for key generation
RUN wget -qO- https://github.com/SagerNet/sing-box/releases/download/v1.8.0/sing-box-1.8.0-linux-amd64.tar.gz | tar xz -C /tmp && \
    mv /tmp/sing-box-1.8.0-linux-amd64/sing-box /usr/local/bin/ && \
    rm -rf /tmp/sing-box-*

WORKDIR /app

COPY --from=builder /app/node-agent .

EXPOSE 8880

CMD ["./node-agent"]
