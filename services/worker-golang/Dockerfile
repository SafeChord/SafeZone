# Build 
FROM golang:1.24-bookworm AS build
WORKDIR /module
COPY services/worker-golang/go.mod services/worker-golang/go.sum /module/
RUN go mod download
COPY services/worker-golang/app /module/app
RUN go build -o worker ./app/main.go

# Deploy 
FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /module/worker .
CMD ["./worker"]