# Identity Provider backend written in Golang
The purpose of this project is to make it possible for anyone to run a basic Identity Provider.

It does *not* have anything to do with OAuth2 in any way, but is meant to be used as the Identity Provider for another service like ORY Hydra (https://github.com/ory/hydra).

This project will only give you the required API endpoints for managing an Identity Provider - no GUI is included. However, it will be able to run hand-in-hand with https://github.com/opensentry/idpui as the graphical web interface.

# Requirements
There is a set of requirements to be met in order to run the Identity Provider.

## Hardware
 * @TODO: Memory requirements
 * @TODO: Network requirements
 * @TODO: Storage requirements

## Software
 * Docker (https://www.docker.com/) or a compatible containerization technology.

## Ports Used
The Identity Provider can be configured to run on any port, using the configuration options.

# Getting started
First of all make sure docker is installed and ready to use.

Next, run the following commands:
```
$ git clone git@github.com:OpenSentry/idp.git
$ cd idp
$ # This will build a docker image by getting all necessary requirements and compiling the go project.
$ docker build -t idp .
$ # When the image has been build, use the following docker command to start it up:
$ docker run -it -p 8080:8080 -v $(pwd):/go/src/idp idp
```

The endpoint documentation is found in the [https://github.com/OpenSentry/idp/wiki](wiki)