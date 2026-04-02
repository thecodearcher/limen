# Build from repository root so local `replace ../../...` paths resolve:
#   docker build -f examples/basic/Dockerfile -t limen-basic .

FROM golang:1.25-alpine AS builder

WORKDIR /src

# CA certs are needed by `go mod download` for HTTPS module fetches.
RUN apk add --no-cache ca-certificates

COPY . .

# Build the example app binary from the workspace.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/basic ./examples/basic

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/basic /app/basic

EXPOSE 8080

CMD ["/app/basic"]
