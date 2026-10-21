# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend (CGO-free SQLite)
FROM golang:1.27-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# Stage 3: Final Runtime Image
FROM alpine:3.20
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
COPY --from=backend-builder /app/server /app/server
COPY --from=frontend-builder /app/frontend/dist /app/static
RUN mkdir -p /app/data

ENV PORT=8080
ENV DB_PATH=/app/data/tracker.db
ENV STATIC_DIR=/app/static

EXPOSE 8080
CMD ["/app/server"]
