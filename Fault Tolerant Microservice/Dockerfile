# Use Go 1.19
FROM golang:1.19

# Set Working Directory
WORKDIR /app

# Copy Go module files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy application code
COPY *.go ./

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./product_search_api

# Execute permisions for binary
RUN chmod +x ./product_search_api

# Expose Port 8080
EXPOSE 8080

# Run the application
CMD [ "./product_search_api" ]