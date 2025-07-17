# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags "-s -w" -o qbr .

# Runtime stage
FROM scratch

COPY --from=builder /app/qbr /qbr

ENTRYPOINT ["/qbr"]
