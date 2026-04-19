FROM golang:1.25-alpine

WORKDIR /app

# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the app to ensure everything is in order
RUN go build -o main .

CMD ["air"]
