# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from golang v1.11 base image
FROM golang:1.12-alpine

# Add Maintainer Info
LABEL maintainer="Lasse Nielsen <65roed@gmail.com>"

RUN apk add --update --no-cache ca-certificates cmake make g++ openssl-dev git curl pkgconfig

# RUN apt install -y libssl1.0.0
RUN git clone -b v1.7.4 https://github.com/neo4j-drivers/seabolt.git /seabolt

# invoke cmake build and install artifacts - default location is /usr/local
WORKDIR /seabolt/build

# CMAKE_INSTALL_LIBDIR=lib is a hack where we override default lib64 to lib to workaround a defect
# in our generated pkg-config file
RUN cmake -D CMAKE_BUILD_TYPE=Release -D CMAKE_INSTALL_LIBDIR=lib .. && cmake --build . --target install

# Set the Current Working Directory inside the container
# WORKDIR $GOPATH/src/github.com/charmixer/idp
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

RUN go build -o idp .

# This container exposes port 443 to the docker network
EXPOSE 443

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

#USER 1000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["idp"]
