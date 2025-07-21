FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ca-certificates upx
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o qbr .
RUN upx --best --lzma qbr

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/qbr /qbr
USER nonroot:nonroot
ENTRYPOINT ["/qbr"]
