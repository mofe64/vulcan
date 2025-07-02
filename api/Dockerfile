# syntax=docker/dockerfile:1.4
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copy the go.work file from the root of your repository (build context)
# It will be copied to /app/go.work inside the container
COPY go.work ./

# Copy the api/go.mod and api/go.sum into the 'api/' directory inside the container.


# Copy both the api and operator directories to the builder stage.
# This will bring in api/go.mod, api/go.sum, and all other source files.
# It also brings in the operator directory needed for the 'replace' directive.
COPY api api/
COPY operator operator/

# Set the GOWORK environment variable to point to the go.work file.
# This ensures that `go` commands executed later respect the workspace.
ENV GOWORK=/app/go.work

# Synchronize modules in the workspace.
# This ensures all modules, including replaced ones, are properly set up.
# This must be run from the workspace root (which is /app)
RUN go work sync

# Build the API binary
# CGO_ENABLED=0 is important for creating statically linked binaries on Alpine.
# -a links all packages statically, -ldflags '-s -w' reduces binary size by removing debug info.
# set the working directory to the directory containing the main function
WORKDIR /app/api/cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w' -o /usr/local/bin/api .

# Final stage: Create a minimal image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /usr/local/bin/api .

# Expose the port API listens on (e.g., 8080)
EXPOSE 8080

# Command to run the API
ENTRYPOINT ["/app/api"]