# Base Go image for building
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum files separately to cache modules if they haven't changed
COPY go.mod go.sum ./

# Download dependencies to cache them if unchanged
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the publisher binary
RUN go build -o /publisher ./boot/publisher/publisher.main.go

# Multi-stage build for the final publisher image
FROM alpine:latest
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /publisher .

# Expose the port as needed for the publisher service
EXPOSE 5000

ENTRYPOINT ["./publisher"]
