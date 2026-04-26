# Build Go app
FROM golang:1.25-alpine
WORKDIR /app
# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .
CMD ["air"]
