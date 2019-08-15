# Identity Provider backend written in Golang
The purpose of this project is to make it possible for anyone to run a very simple identity provider.

It does *not* have anything to do with OAuth2 in any way, but is meant to be used as the Identity Provider for another service like ORY Hydra (https://github.com/ory/hydra).

This project will only give you the required API endpoints for managing an Identity Provider - no GUI is included. However, it will be able to run hand-in-hand with https://github.com/charmixer/golang-idp-fe as the graphical web interface.

Table of Contents
=================

  * [Getting started](#getting-started)
  * [API documentation](#api-documentation)  
    * [Concepts](#concepts)
      * [Identity](#identity)    
    * [Endpoints](#endpoints)
    * [Create an Identity](#create-an-identity)
    * [Change a Password](#change-a-password)    

# Getting started
First of all make sure docker is installed and ready to use.

Next, run the following commands:
```
$ git clone git@github.com:CharMixer/golang-idp-be.git
$ cd golang-idp-be
$ # This will build a docker image by getting all necessary requirements and compiling the go project.
$ docker build -t idpapi .
$ # When the image has been build, use the following docker command to start it up:
$ docker run -it -p 8080:8080 -v $(pwd):/go/src/golang-idp-be idpapi
```

Note that the default settings is a development build, which can be used for automatic rebuilding of the go code with the help of https://github.com/pilu/fresh. Later on the environment variable `APP_ENV` will be used to define a production or development environment

# API documentation

The idpapi exposes a set of endpoints that can be used to control identities.

## Concepts

### Identity
An identity is a representation of a person, an app or anything that needs to be uniquely identified within the system

```json
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
All endpoints can only be reached trough HTTPS with TLS. All endpoints are protected by OAuth2 scopes that are required by the client to call the endpoints. The following endpoints are exposed:

```json
{
  "/identities": {
    "description": "CRUD operations on the collection of identities",
    "method": {
      "get": {
        "description": "Read the data stored for an Identity",
        "required_scope": "idpapi.identities.get"
      },
      "post": {
        "description": "Create a new Identity",
        "required_scope": "idpapi.identities.post"
      },
      "put": {
        "description": "Update data stored for an Identity",
        "required_scope": "idpapi.identities.put"
      },
      "delete": {
        "description": "Not implemented",
        "required_scope": "idpapi.identities.delete"
      }
    }    
  },
  "/identities/authenticate": {
    "description": "Use to authenticate an identity",
    "method": {      
      "post": {
        "description": "Authenticate an Identity",
        "required_scope": "idpapi.authenticate"
      }      
    }   
  },
  "/identities/password": {
    "description": "
      Use to change the password of an identity.
      Password is not part of CRUD on /identities because password is the primary concern
      of protection, hence treated as a first class citizen in the system.
    ",
    "method": {      
      "post": {
        "description": "Change password for an Identity",
        "required_scope": "idpapi.authenticate"
      }      
    }
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

## Create an Identity
To create a new identity a `POST` request must be made to the `/identities` endpoint. Specifying an `id` for the Identity, a name, email and an optional `password` in plain text. Hashing of the password will be done by the endpoint, before sending it to storage. The hashing algorithm is performed by the bcrypt library `golang.org/x/crypto/bcrypt` using the following function:

```golang
func CreatePassword(password string) (string, error) {
  hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
  if err != nil {
    return "", err
  }
  return string(hash), nil
}
```

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idpapi.identities.post` to call the endpoint or the request will be denied.

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"id":"test", "password":"secret", "name":"Test", "email":"test@domain.com"}'
     -X POST https://id.domain.com/api/identities
```

## Change a Password
To change the password of an identity a `POST` request must be made to the `/identities/password` endpoint. To ensure that password handling is not taken lightly but rather considered a first class element in the system. It has received its own endpoint. This ensures separation of concerns and hopefully help to prevent accidental updates when updating an Identity with other data.

The change password endpoint is using the same bcrypt library algorithm as when creating an Identity with a password. See [Create an Identity](#create-an-identity).

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"id":"test", "password":"anewsecret"}'
     -X POST https://id.domain.com/api/identities/password
```
