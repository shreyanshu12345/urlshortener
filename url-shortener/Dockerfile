# ---------- Build Stage ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# ---------- Runtime Stage ----------
FROM alpine:latest

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

# Copy templates folder
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./main"]