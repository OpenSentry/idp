# Identity Provider backend written in Golang
The purpose of this project is to make it possible for anyone to run a very simple identity provider.

It does *not* have anything to do with OAuth2 in any way, but is meant to be used as the Identity Provider for another service like ORY Hydra (https://github.com/ory/hydra).

This project will only give you the required API endpoints for managing an Identity Provider - no GUI is included. However, it will be able to run hand-in-hand with https://github.com/linuxdk/golang-idp-fe as the graphical web interface.

# Getting started
First of all make sure docker is installed and ready to use.

Next, run the following commands:
```
$ git clone https://github.com/linuxdk/golang-idp-be.git
$ cd golang-idp-be
$ # This will build a docker image by getting all necessary requirements and compiling the go project.
$ docker build -t idp .
$ # When the image has been build, use the following docker command to start it up:
$ docker run -it -p 8080:8080 -v $(pwd):/go/src/idp idp
```

Note that the default settings is a development build, which can be used for automatic rebuilding of the go code with the help of https://github.com/pilu/fresh. Later on the environment variable `APP_ENV` will be used to define a production or development environment
