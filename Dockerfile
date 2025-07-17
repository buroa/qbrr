FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=0
RUN go build \
        -a -installsuffix cgo \
        -ldflags "-s -w -extldflags '-static'" \
        -trimpath \
        -tags netgo \
        -o qbr .

FROM scratch
COPY --from=builder /app/qbr /qbr
ENTRYPOINT ["/qbr"]
