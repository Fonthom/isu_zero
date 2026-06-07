FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o isu_zero ./cmd/server
 
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/isu_zero .
EXPOSE 8080
CMD ["./isu_zero"]
 