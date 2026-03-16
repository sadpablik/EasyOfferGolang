# Build stage
FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Generate Swagger docs during image build and fail fast if expected routes are missing.
RUN rm -rf docs \
	&& go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main.go -o docs --parseInternal \
	&& grep -q '"/api/v1/interviews/start"' docs/swagger.json \
	&& grep -q '"/api/v1/interviews/{id}/answer"' docs/swagger.json

ENV CGO_ENABLED=0
RUN go build -o main ./cmd

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]