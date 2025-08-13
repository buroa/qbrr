FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates upx
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/qbrr cmd/qbrr/main.go
RUN upx --best --lzma bin/qbrr

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/bin/qbrr /qbrr
USER nonroot:nonroot
ENTRYPOINT ["/qbrr"]
