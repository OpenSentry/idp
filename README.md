# Identity Provider backend written in Golang
The purpose of this project is to make it possible for anyone to run a very simple identity provider.

It does *not* have anything to do with OAuth2 in any way, but is meant to be used as the Identity Provider for another service like ORY Hydra (https://github.com/ory/hydra).

This project will only give you the required API endpoints for managing an Identity Provider - no GUI is included. However, it will be able to run hand-in-hand with https://github.com/charmixer/golang-idp-fe as the graphical web interface.

# Getting started
First of all make sure docker is installed and ready to use.

Next, run the following commands:
```
$ git clone git@github.com:CharMixer/golang-idp-be.git
$ cd golang-idp-be
$ # This will build a docker image by getting all necessary requirements and compiling the go project.
$ docker build -t idp-be .
$ # When the image has been build, use the following docker command to start it up:
$ docker run -it -p 8080:8080 -v $(pwd):/go/src/golang-idp-be idp-be
```

Note that the default settings is a development build, which can be used for automatic rebuilding of the go code with the help of https://github.com/pilu/fresh. Later on the environment variable `APP_ENV` will be used to define a production or development environment

# API documentation

The idpapi exposes a set of endpoints that can be used to control identities.

## Concepts

### Identity
An identity is a representation of a person, an app or anything that needs to be uniquely identified within the system

#### Fields
```JSON
{
  "id": {
    "type": "string",
    "required": 1,
    "description": "A globally unique identifier. An example could be a unix username"
  },
  "password": {
    "type": "string",
    "required": 0,
    "description": "A password hash. Please do not store plain text passwords!"
  },
  "name": {
    "type": "string",
    "required": 0,
    "description": "The name used to address the identity"
  },
  "email": {
    "type": "string",
    "required": 0,
    "description": "The email where the identity can be reached for password reset etc."
  }
}
```

## Endpoints
The idpapi exposes the following endpoints

```JSON
{
  "/identities": {
    "description": "CRUD operations on the collection of identities"
  },
  "/identities/authenticate": {
    "description": "Use to authenticate an identity"
  },
  "/identities/password": {
    "description": "
      Use to change the password of an identity.
      Password is not part of CRUD on /identities because password is the primary concern of protection,
      hence treated as a first class citizen in the system.
    "
  },
  "/identities/logout": {
    "description": "Use to logout an identity"
  },
  "/identities/revoke": {
    "description": "Use to revoke an identity. See delete on /identities"
  },    
  "/identities/recover": {
    "description": "Use to recover an identity if password was forgotten"
  }
}
```
