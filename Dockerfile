# Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install --legacy-peer-deps
COPY frontend/ ./
RUN npm run build-only

# Build Go app
FROM golang:1.25-alpine
WORKDIR /app
# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend from previous stage
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN go build -o main .
CMD ["air"]
