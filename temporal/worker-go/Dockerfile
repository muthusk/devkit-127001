# =============================================================================
# Go Worker — Multi-stage Docker build
# =============================================================================

# --- Build stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy module files and source
COPY . .

# Download dependencies (generates go.sum if missing)
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./main.go

# --- Runtime stage ---
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /worker /worker

ENTRYPOINT ["/worker"]
