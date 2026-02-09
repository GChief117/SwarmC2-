# Stage 1: Build Frontend
FROM node:18-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
ARG VITE_MAPTILER_KEY
ENV VITE_MAPTILER_KEY=$VITE_MAPTILER_KEY
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.21-alpine AS backend-build
WORKDIR /app/backend
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
RUN go mod tidy
# Copy frontend dist into backend/static
COPY --from=frontend-build /app/frontend/dist ./static/
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 3: Production
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=backend-build /app/backend/server .
COPY --from=backend-build /app/backend/static ./static/
EXPOSE 8080
CMD ["./server"]
