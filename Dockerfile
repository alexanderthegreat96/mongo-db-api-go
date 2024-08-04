# Create a lightweight image to run the binary
FROM debian:latest

# Set the working directory for the runtime image
RUN mkdir -p /app/mongo-api
WORKDIR /app/mongo-api

# Copy the binary from the builder image
COPY . /app/mongo-api

RUN chmod +x mongo-api

# Command to run the executable
EXPOSE 9874
ENTRYPOINT ["./mongo-api"]