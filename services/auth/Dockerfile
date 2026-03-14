# Build stage
FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Generate Swagger docs during image build so /swagger works without manual swag init.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
RUN rm -rf docs
RUN /go/bin/swag init -g main.go -d ./cmd,./api/handlers --parseInternal --outputTypes json
# Disable CGO to produce a statically linked binary (works well on alpine)
ENV CGO_ENABLED=0
RUN go build -o main ./cmd

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
CMD ["./main"]