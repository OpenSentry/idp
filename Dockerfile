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
WORKDIR $GOPATH/src/idp

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download all the dependencies
# https://stackoverflow.com/questions/28031603/what-do-three-dots-mean-in-go-command-line-invocations
RUN go get github.com/gin-gonic/gin 
RUN go get github.com/neo4j/neo4j-go-driver/neo4j 

# Install the package
RUN go install -v ./...

# This container exposes port 8080 to the outside world
EXPOSE 8080

CMD if [ "${APP_ENV}" = "production" ]; \
	then \
	  idp; \
	else \
	  go get github.com/pilu/fresh && \
	  fresh; \
	fi

# DEVELOPMENT
# RUN export PATH=$PATH:$GOPATH/bin
# RUN go get github.com/gin-gonic/gin 
# CMD ["gin run main.go"]

# PRODUCTION
# Run the executable
# CMD ["idp"]
