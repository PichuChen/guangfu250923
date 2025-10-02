# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /app/guangfu250923 ./cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/guangfu250923 .

# Copy OpenAPI spec (optional, for documentation)
COPY openapi.yaml .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./guangfu250923"]
