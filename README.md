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
    * [A note on scopes](#a-note-on-scopes)    
    * [Endpoints](#endpoints)
      * [POST /identities](#post-identities)
      * [GET /identities](#get-identities)
      * [PUT /identities](#put-identities)
      * [DELETE /identities](#delete-identities)        
      * [POST /identities/deleteverification](#post-identitiesdeleteverification)
      * [POST /identities/authenticate](#post-identitiesauthenticate)          
      * [POST /identities/password](#post-identitiespassword)
      * [POST /identities/recover](#post-identitiesrecover)
      * [POST /identities/recoververification](#post-identitiesrecoververification)
      * [POST /identities/passcode](#post-identitiespasscode)
      * [POST /identities/2fa](#post-identities2fa)
      * [POST /identities/logout](#post-identitieslogout)
    * [Create an Identity](#create-an-identity)
    * [Change a Password](#change-a-password)    
    * [Authenticate an Identity](#authenticate-an-identity)

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
An identity is a representation of a person, an app or anything that needs to be uniquely identified within a system

```json
{
  "id": {
    "type": "string",    
    "description": "A globally unique identifier. An example could be a unix username"
  },
  "password": {
    "type": "string",    
    "description": "A password hash. Please do not store plain text passwords!"
  },
  "name": {
    "type": "string",    
    "description": "The name used to address the identity"
  },
  "email": {
    "type": "string",    
    "description": "The email where the identity can be reached for password reset etc."
  }
}
```

## A note on scopes
The scope `authenticate:identity` is used whenever the password credentials of an Identity is involved. This also include verification codes that are a form of two-factor alias for passwords. This scope should be restricted to applications inside the trust zone only.

## Endpoints
All endpoints can only be reached trough HTTPS with TLS. All endpoints are protected by OAuth2 scopes that are required by the client to call the endpoints. The following endpoints are exposed:

### POST /identities

Create an Identity. Requires scope `authenticate:identity`

#### Input
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": false
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": true
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

### GET /identities

Read an Identity. Requires scope `read:identity`

#### Input
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": false
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": true
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

### PUT /identities

Update an Identity. Requires scope `update:identity`. Note that it is not possible to update the password or other password or code credentials on the Identity using this function. This is to prevent accidental updates and to seperate what functions can be exposed to UI applications outside the trust zone.

#### Input
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": false
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": true
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

### DELETE /identities

Update an Identity. Requires scope `delete:identity`. This will send an email with a verification code to the email of the Identity. The verification code should be used with the endpoint [POST /identities/deleteverification](#delete-identitiesdeleteverification).

#### Input
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": false
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "password": {
     "type": "string",
     "required": true
  },
  "name": {
     "type": "string",
     "required": false
  },
  "email": {
     "type": "string",
     "required": false
  }
}
```

### POST /identities/deleteverification

Confirm deletion of an Identity. Requires scope `delete:identity`. If the verification code matches what was generated by the [DELETE /identities](#delete-identities) endpoint. The Identity will be deleted. Beware this is an unrecoverable action. Use with care.

#### Input
```json
{
  "id": {
     "type": "string",
     "required": true
   },
  "verification_code": {
     "type": "string",
     "required": true
  },
  "redirect_to": {
     "type": "string",
     "required": true
  }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "required": true
  },
  "verified": {
     "type": "bool",
     "required": true
  },
  "redirect_to": {
     "type": "string",
     "required": true
  }
}
```

### POST /identities/authenticate

Authenticate an Identity. Requires scope `authenticate:identity`. This will validate credentials of a user to match it to an Identity.

#### Input
```json
{
  "challenge": {
    "type": "string",
    "required": true
  },
  "id": {
    "type": "string",
    "required": false
  },
  "password": {
    "type": "string",
    "required": false
  },
  "password": {
    "type": "string",
    "required": false
  },

}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  },
  "not_found": {
    "type": "bool",
    "required": true
  },
  "authenticated": {
    "type": "bool",
    "required": true
  },
  "require_2fa": {
    "type": "bool",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  }
}
```

### POST /identities/password

Update the password of an Identity. Requires scope `authenticate:identity`. This overrides any currently registered password. Use with care.

#### Input
```json
{  
  "id": {
    "type": "string",
    "required": true
  },
  "password": {
    "type": "string",
    "required": true
  }  
}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  }  
}
```

### POST /identities/recover

Initiate the process of recovering an Identity. Requires scope `recover:identity`. This will send an email with a verification code to the email of the Identity. The verification code should be used with the endpoint [POST /identities/recoververification](#post-identitiesrecoververification).

#### Input
```json
{  
  "id": {
    "type": "string",
    "required": true
  }  
}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  }  
}
```

### POST /identities/recoververification

Confirm recovery of an Identity. Requires scope `authenticate:identity`. If the verification code matches what was generated by the [POST /identities/recover](#post-identitiesrecover) endpoint. The password of the Identity will be updated. Use with care.

#### Input
```json
{  
  "id": {
    "type": "string",
    "required": true
  },  
  "verification_code": {
    "type": "string",
    "required": true
  },
  "password": {
    "type": "string",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  },
}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  },
  "verified": {
    "type": "bool",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  }  
}
```

### POST /identities/passcode

Confirm two-factor code of an Identity. Requires scope `authenticate:identity`. If the verification code matches what was generated by the [POST /identities/2fa](#post-identities2fa) endpoint. The Identity is authenticated.

#### Input
```json
{  
  "id": {
    "type": "string",
    "required": true
  },  
  "passcode": {
    "type": "string",
    "required": true
  },
  "challenge": {
    "type": "string",
    "required": true
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  },
  "verified": {
    "type": "bool",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  }  
}
```

### POST /identities/2fa

Enable or disable two-factor authentication for an Identity. Requires scope `authenticate:identity`.

#### Input
```json
{  
  "id": {
    "type": "string",
    "required": true
  },  
  "require_2fa": {
    "type": "bool",
    "required": true
  },
  "secret_2fa": {
    "type": "string",
    "required": true
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",
    "required": true
  }
}
```

### POST /identities/logout

Log an Identity out of the session. Requires scope `authenticate:identity`.

#### Input
```json
{  
  "challenge": {
    "type": "string",
    "required": true
  }
}
```

#### Output
```json
{
  "redirect_to": {
    "type": "string",
    "required": true
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
To change the password of an identity a `POST` request must be made to the `/identities/password` endpoint.

To ensure that password handling is not taken lightly but rather considered a first class element in the system. It has received its own endpoint. This ensures separation of concerns and hopefully help to prevent accidental updates when updating an Identity with other data.

The change password endpoint is using the same bcrypt library algorithm as when creating an Identity with a password. See [Create an Identity](#create-an-identity).

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idpapi.authenticate` to call the endpoint or the request will be denied.

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"id":"test", "password":"anewsecret"}'
     -X POST https://id.domain.com/api/identities/password
```

## Authenticate an Identity
To authenticate an Identity, also known as performing a login/signin a `POST` request must be made to the `/identities/authenticate` endpoint. A challenge is required to perform a login. The challenge is obtained by asking Hydra for it when starting the OAuth2 Authorization code flow.

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idpapi.authenticate` to call the endpoint or the request will be denied.

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"challenge":"QWRGvqe56wyega5", "id": "test, ""password":"secret"}'
     -X POST https://id.domain.com/api/identities/authenticate
```
