# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

# Stage 2: Run
FROM alpine:3.19

WORKDIR /app

# CA certs needed for HTTPS redirects
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server .

EXPOSE 8080 9090

CMD ["./server"]