# Use Go 1.21 as base image
FROM golang:1.21-bullseye

# Install pandoc
RUN apt-get update && apt-get install -y pandoc && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o wiki-server

# Expose port 8080
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["./wiki-server"]
