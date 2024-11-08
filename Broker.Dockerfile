FROM golang:1.20-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY boot/broker/ ./broker/

RUN go build -o /broker ./broker/main.go

# Stage 2: Run
FROM alpine:latest
WORKDIR /root/


COPY --from=builder /broker .

EXPOSE 50051

ENTRYPOINT ["./broker"]
