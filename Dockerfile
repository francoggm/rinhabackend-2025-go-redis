# Builder stages
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Optimizing the binary omitting debug information with the flags -w and -s
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server/main.go

# Runner stages
FROM scratch

COPY --from=builder /app/server server

EXPOSE 8080

CMD ["/server"]
