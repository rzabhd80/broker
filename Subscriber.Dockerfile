
FROM golang:1.20-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY boot/subscriber/ ./subscriber/

RUN go build -o /subscriber ./subscriber/main.go

# Stage 2: Run
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /subscriber .

EXPOSE 50053

ENTRYPOINT ["./subscriber"]
