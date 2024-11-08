# Multi-stage build for the publisher service
# Stage 1: Build
FROM golang:1.20-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum files to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the publisher service code
COPY boot/publisher/ ./publisher/

# Build the publisher binary
RUN go build -o /publisher ./publisher/main.go

# Stage 2: Run
FROM alpine:latest
WORKDIR /root/

# Copy the built binary from builder stage
COPY --from=builder /publisher .

# Expose port (adjust as needed)
EXPOSE 50052

# Run the publisher binary
ENTRYPOINT ["./publisher"]
