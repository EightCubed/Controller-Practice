# Use Golang for building the binary
FROM golang:1.21 AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the binary, specifying the `cmd` directory
RUN go build -o /app/controller ./cmd/main.go

# Use a minimal base image for the final container
FROM alpine:latest

WORKDIR /app

# Copy the built binary
COPY --from=builder /app/controller /app/controller

# Ensure the binary is executable
RUN chmod +x /app/controller

CMD ["/app/controller"]
