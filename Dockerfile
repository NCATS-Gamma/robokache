FROM golang:1.17.8-bullseye

# Add Maintainer Info
LABEL org.opencontainers.image.source https://github.com/NCATS-Gamma/robokache

# Set the Current Working Directory inside the container
WORKDIR /app

# make sure all is writeable for the nru USER later on
RUN chmod -R 777 .
RUN mkdir /home/nru && \
    chmod -R 777 /home/nru

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# create a new user and use it.
RUN useradd -M -u 1001 nru
USER nru

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main cmd/main.go

# Set Gin to run in release mode
ENV GIN_MODE release

# Set executable as default command
CMD ["./main"]
