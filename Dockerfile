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
WORKDIR $GOPATH/src/golang-idp-be

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download all the dependencies
# https://stackoverflow.com/questions/28031603/what-do-three-dots-mean-in-go-command-line-invocations
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

# This container exposes port 443 to the docker network
EXPOSE 443

USER 1000

ARG release_build=1
ENV release_build=$release_build

CMD update-ca-certificates && if [ "$release_build" = "1" ]; \
      then \
        golang-idp-be; \
      else \
        go get github.com/pilu/fresh && \
        fresh; \
      fi
