# Build stage
FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Generate Swagger docs during image build.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
RUN rm -rf docs
RUN /go/bin/swag init -g main.go -d ./cmd --parseInternal -o ./docs

ENV CGO_ENABLED=0
RUN go build -o main ./cmd

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]