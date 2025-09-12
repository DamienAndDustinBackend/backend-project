FROM golang:1.24.5

WORKDIR /backend-project

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy code into the image
COPY ./ ./

# Compile
RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-backend-project

# Open port 8080
EXPOSE 8080

# Run 
CMD ["/docker-backend-project"]
