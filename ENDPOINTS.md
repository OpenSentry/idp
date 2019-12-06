# Endpoints

Endpoints exposed by the Identity Provider service. Hence forth named IDP. The endpoints control anything in relations to authenticating an Identity.

Table of Contents
=================

  * [Usage](#usage)
  * [Concepts](#concepts)
    * [Identity](#identity)
    * [Human](#human)                   
    * [Challenge](#challenge)
  * [Endpoints](#endpoints)        
    * [GET /identities](#get-identities)

    * [POST /humans](#post-humans)
    * [GET /humans](#get-humans)
    * [PUT /humans](#put-humans)
    * [DELETE /humans](#delete-humans)
    * [PUT /humans/deleteverification](#put-humansdeleteverification)
    * [POST /humans/authenticate](#post-humansauthenticate)          
    * [PUT /humans/password](#put-humanspassword)
    * [POST /humans/recover](#post-humansrecover)
    * [PUT /humans/recoververification](#put-humansrecoververification)
    * [PUT /humans/totp](#put-humanstotp)      
    * [PUT /humans/email](#put-humansemail)
    * [POST /humans/emailchange](#post-humansemailchange)
    * [PUT /humans/emailchange](#put-humansemailchange)
    * [GET /humans/logout](#get-humanslogout)
    * [POST /humans/logout](#post-humanslogout)
    * [PUT /humans/logout](#put-humanslogout)

    * [GET /challenges](#get-challenges)
    * [POST /challenges](#post-challenges)
    * [POST /challenges/verify](post-challengesverify)
  * [Scopes](#scopes)
    * [A note on scopes](#a-note-on-scopes)   
  * [Create an Identity](#create-an-identity)
  * [Change a Password](#change-a-password)    
  * [Authenticate an Identity](#authenticate-an-identity)

## Usage

The functions in this REST API is using HTTP method POST to allow for a uniform interface on all endpoints and overcome the inconsistencies in HTTP GET vs POST. To use a GET, POST, PUT or DELETE you must set the X-HTTP-METHOD-OVERRIDE header.

All endpoints can only be reached trough HTTPS with TLS. All endpoints are protected by OAuth2 scopes that are required by the client to call the endpoints.

## Concepts

### Identity
`Endpoint: /identities`

An identity is a representation of a person, an application or anything that needs to be uniquely identified within a system.

```json
{
  "id": {
    "type": "string",    
    "description": "A globally unique identifier.",
    "validate": "uuid"
  }
}
```

An Identity is composed into a more specific type using labels such as `Human`, `Client`, `ResourceServer`, `Invite` and `Role`. Each label indicates that more data is available at the endpoint acting on the Identity type.

#### Human
`Endpoint: /humans`

```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the person in the system.",
    "validate": "uuid, unique"
  },
  "password": {
    "type": "string",
    "description": "Hash of the password used to authenticate the person.",
    "validate": "bcrypt"    
  },    
  "email": {
    "type": "string",
    "description": "Email claimed by the person. Considered an alias to id.",
    "validate": "email, unique"
  },    
  "username": {
    "type": "string",
    "description": "Username chosen by the person. Considered an alias to id.",    
    "validate": "unique"
  },
  "name": {
    "type": "string",
    "description": "The name of the person."
  },
  "totp_required": {
    "type": "bool",
    "description": "Flag defining if a person must authenticate using Timed One Time Password algorithm."
  },
  "totp_secret": {
    "type": "string",
    "description": "Encrypted secret used to authenticate the person using Timed One Time Password algorithm.",    
  },
  "allow_login": {
    "type": "bool",
    "description": "Flag defining if the person is allowed to perform authentication at all.",    
  },
  "email_confirmed_at": {
    "type": "int64",
    "description": "Time of email confirmation in unixtime."
  }
}
```

### Challenge
`Endpoint: /challenges`

A challenge is a question prompted to answer one of these two questions:
 * Who are you?
 * Do you have access to this resource?

It is used as a security measure, when changing recovery method or when two-factor authenticating an Identity.

```json
{
  "otp_challenge": {
    "type": "string",
    "description": "A globally unique identifier",
    "validate": "uuid",
  }
}
```


### GET /identities

Read an Identity. Requires scope `idp:read:identities`.

#### Input
```json
{
  "id": {
     "type": "string",
     "description": "A globally unique identifier.",
     "validate": "optional, uuid, required_without=search"     
   },
   "search": {
     "type": "string",
     "description": "The username or email of the identity",
     "validate": "required_without=id"
   }
}
```

#### Output
```json
{
  "id": {
     "type": "string",
     "description": "The globally unique identifier for the Identity.",
     "validate": "required, uuid"     
   },
   "labels": {
     "type": "array of string",
     "description": "Labels denoting types like Human, Client, Invite and Resource Provider etc.",
     "validate": "optional"
   }
}
```


### POST /humans

Create a human. Requires scope `idp:create:humans`.

#### Input
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "uuid, unique"
  },
  "password": {
    "type": "string",
    "description": "Cleartext password entered by the human.",
    "validate": "required, max=55"
  },
  "username": {
    "type": "string",
    "description": "Username entered by the human.",
    "validate": "optional"
  },
  "email": {
    "type": "string",
    "description": "The confirmed email of the human.",
    "validate": "optional, email",
  },
  "name": {
    "type": "string",
    "description": "The name of the human.",
    "validate": "optional"
  },
  "allow_login": {
    "type": "bool",
    "description": "Flag defining if the person is allowed to perform authentication at all."
  },
  "email_confirmed_at": {
    "type": "int64",
    "description": "Time of email confirmation in unixtime.",
    "validate": "optional"
  }
}
```

#### Output
See [Human](#human) definition. This endpoint is the only endpoint that will return the password hash of the person.


### GET /humans
Read data of a human. Requires scope `idp:read:humans`.

#### Input
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "optional, uuid"
  },
  "email": {
    "type": "string",
    "description": "Email claimed by the human.",
    "validate": "optional, email"
  },
  "username": {
    "type": "string",
    "description": "Username chosen by the human.",
    "validate": "optional"
  }
}
```

#### Output
See [Human](#human) definition.


### PUT /humans

Update human attributes that are non vital to the security of the authentication process. Requires scope `idp:update:humans`.

This endpoints has limitations on which part of the human model it is allowed to update for security reasons. This means that anything related to passwords, otp codes and recovery of identity is not allowed to be updated by this endpoint. Instead named endpoints for this exists. Structuring the endpoint like this prevents accidental updates that compromise the security of the authentication processes. This also means that the endpoint can safely be exposed to third party applications outside the trust zone iff one should wish it.

#### Input
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },  
  "name": {
    "type": "string",
    "description": "The name of the person."
  }
}
```

#### Output
See [Human](#human) definition.


### DELETE /humans

Request deletion of an human. Requires scope `idp:delete:humans`.

This endpoints starts the process of erasing a human identity from the system. Since this is a non recoverable action a challenge of deletion is issued which must be confirmed by the human in question. The challenge is sent to the email registered to the human.

The challenge should be verified using this endpoint [PUT /humans/deleteverification](#put-humansdeleteverification).

#### Input
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when deletion succeeds",
    "validate": "required, uri"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url to start the deletion challenge process",
    "validate": "required, uri"
  }
}
```

### PUT /humans/deleteverification

Delete a human. Requires scope `idp:update:humans:deleteverification`.

If the challenge is a deletion challenge and it is verified the human will be erased from the system. The action is non recoverable, use with care.

#### Input
```json
{
  "delete_challenge": {
    "type": "string",
    "description": "The identifier for the challenge in the system.",
    "validate": "required, uuid"
  },  
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when deletion succeeded.",
    "validate": "required, uri"
  },
  "verified": {
     "type": "bool",
     "description": "Flag indication if the challenge was verified successfully or not."
  }
}
```


### POST /humans/authenticate

Authenticate a human. Requires scope `idp:create:humans:authenticate`.

This will validate credentials provided by the human and match them to the human identity in the system.

#### Input
```json
{
  "challenge": {
    "type": "string",
    "description": "The identifier for the login challenge in the system.",
    "validate": "required, uuid"
  },
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "optional, uuid"
  },
  "password": {
    "type": "string",
    "description": "Cleartext password entered by the human.",
    "validate": "optional, max=55"
  },
  "otp_challenge": {
    "type": "string",
    "description": "The identifier for the OTP challenge in the system.",
    "validate": "optional, uuid"
  },
  "email_challenge": {
    "type": "string",
    "description": "The identifier for the email challenge in the system.",
    "validate": "optional, uuid"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "optional, uuid"
  },
  "authenticated": {
    "type": "bool",
    "description": "Flag indicating if human authenticated",
    "validate": "required"
  },  
  "totp_required": {
    "type": "bool",
    "description": "Flag indicating that human requires OTP authentication.",
    "validate": "required"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when authentication succeeded.",
    "validate": "optional, uri"
  },
  "is_password_invalid": {
    "type": "bool",    
    "description": "Flag indication that the human exists but password was incorrect.",
    "validate": "required"
  },
  "identity_exists": {
    "type": "bool",
    "description": "Flag indication that the human does not exist.",
    "validate": "required"
  },
}
```


### PUT /humans/password

Update the password of a human. Requires scope `idp:update:humans:password`.

This will override the currently registered password for the human. Use with care.

#### Input
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "password": {
    "type": "string",
    "description": "Cleartext password entered by the human.",
    "validate": "required, max=55"
  }
}
```

#### Output
See [Human](#human) definition


### POST /humans/recover

Recover a human identity. Requires scope `idp:create:humans:recover`.

This endpoints starts the process of recovering a human identity from the system. This will issue an authentication challenge via email to the human. The challenge is sent to the email registered to the human.

The challenge should be verified using this endpoint [PUT /humans/recoververification](#put-humansrecoververification).

#### Input
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url to start the recovery process.",
    "validate": "required, uri"
  },
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when recover succeeded.",
    "validate": "required, uri"
  },
  "verified": {
     "type": "bool",
     "description": "Flag indication if the challenge was verified successfully or not."
  }
}
```


### PUT /humans/recoververification

Confirm recovery of a human identity. Requires scope `idp:update:humans:recoververification`.

If the challenge is a recover challenge and it is verified the password registered to the human will be updated to match the newly entered. Use with care.

#### Input
```json
{    
  "recover_challenge": {
    "type": "string",
    "description": "The identifier for the challenge in the system.",
    "validate": "required, uuid"
  },  
  "new_password": {
    "type": "string",
    "description": "Cleartext password entered by the human.",
    "validate": "required, max=55"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when recover succeeded.",
    "validate": "required, uri"
  },
  "verified": {
     "type": "bool",
     "description": "Flag indication if the challenge was verified successfully or not."
  }
}
```


### PUT /humans/totp

Enable or disable TOTP authentication (2fa) for the human. Requires scope `idp:update:humans:totp`.

Before using this endpoint please ensure that the secret stored in the humans Authenticator App and the secret to be stored by this endpoints agree on the codes by requiring the human to enter at least one code that validates against the secret. If not there is a risk of bricking the identity until a recover process that allows for disabling of TOTP is build into the system.

#### Input
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "totp_required": {
    "type": "bool",    
    "description": "The identifier for the human in the system.",
    "validate": "required"
  },
  "totp_secret": {
    "type": "string",
    "description": "Cleartext TOTP secret.",
    "validate": "required"
  }
}
```

#### Output
See [Human](#human) definition.


### PUT /humans/email

Update email of human. Requires scope `idp:update:humans:email`.

This endpoint updates the email registered to the human. This should only be called after the process of email change process with email confirm challenge has been successfully executed.

#### Input
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "email": {
    "type": "string",    
    "description": "Email to register for human.",
    "validate": "required, email"
  }
}
```

#### Output
See [Human](#human) definition.


### POST /humans/emailchange

Change email of human. Requires scope `idp:create:humans:emailchange`.

#### Input
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "email": {
    "type": "string",
    "description": "Email to register for human.",
    "validate": "required email"
  },  
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url when change succeededs.",
    "validate": "required, uri"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url to start the email change process.",
    "validate": "required, uri"
  }
}
```

### PUT /humans/emailchange

Update email upon challenge verification. Requires scope `idp:update:humans:emailchange`

#### Input
```json
{  
  "email_challenge": {
    "type": "string",    
    "description": "The email challenge identifier in the system.",
    "validate": "required, uuid"
  },
  "email": {
    "type": "string",
    "description": "The email to update to if challenge is verified.",
    "validate": "required, email"
  }
}
```

#### Output
```json
{  
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url after email change succeeds.",
    "validate": "required, uri"
  },
  "verified": {
     "type": "bool",
     "description": "Flag indication if the challenge was verified successfully or not."
  }
}
```


### GET /humans/logout

Read data registered to a logout challenge. Requires scope `idp:read:humans:logout`.

#### Input
```json
{  
  "challenge": {
    "type": "string",    
    "description": "The identifier for the logout challenge in the system.",
    "validate": "required"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "sid": {
    "type": "string",    
    "description": "Session identifier in the system.",    
  },
  "rp_initiated": {
    "type": "string",    
    "description": "Flag indicating wether logout was initiated by a relaying party or not.",
  },
  "request_url": {
    "type": "string",    
    "description": "The url that requested the logout",
    "validate": "required, uri"
  }
}
```

### POST /humans/logout

Create a logout request. Requires scope `idp:create:humans:logout`.

#### Input
```json
{
  "id_token": {
    "type": "string",    
    "description": "Id-token to logout.",
    "validate": "required"
  },  
  "state": {
    "type": "string",    
    "description": "State parameter to prevent redirect CSRF.",
    "validate": "required"
  },  
  "redirect_to": {
    "type": "string",
    "description": "Redirect to url after logout succeeds.",
    "validate": "required, uri"
  }
}
```

#### Output
```json
{
  "redirect_to": {
    "type": "string",
    "description": "Redirect url to start the logout process.",
    "validate": "required, uri"
  }
}
```

### PUT /humans/logout

Accept a logout request. Requires scope `idp:update:humans:logout`.

#### Input
```json
{
  "challenge": {
    "type": "string",    
    "description": "The identifier for the logout challenge in the system.",
    "validate": "required"
  }
}
```

#### Output
```json
{
  "id": {
    "type": "string",    
    "description": "The identifier for the human in the system.",
    "validate": "required, uuid"
  },
  "redirect_to": {
    "type": "string",
    "description": "Redirect url to finalize the logout process.",
    "validate": "required, uri"
  }
}
```


### GET /challenges

Read a challenge. Requires scope `idp:read:challenges`.

#### Input
```json
{
  "otp_challenge": {
    "type": "string",
    "description": "The unique identifier for the challenge",
    "validate": "required, uuid"
  }  
}
```

#### Output
```json
{
  "otp_challenge": {
    "type": "string",
    "description": "The unique identifier for the challenge",
    "validate": "required, uuid"
  },
  "confirmation_type": {
    "type": "int",
    "description": "Type of challenge. Used to decide what communication to prompt the Identity",
    "validate": "numeric"
  },
  "sub": {
    "type": "string",
    "description": "The subject for which this challenge is requested. See OAuth2 for details",
    "validate": "required, uuid"
  },
  "aud": {
    "type": "string",    
    "description": "Intended audience for the challenge. See OAuth2 for details",
    "validate": "required"
  },
  "iat": {
    "type": "int64",    
    "description": "Time of creation for the challenge in unixtime",
    "validate": "required"
  },
  "exp": {
    "type": "int64",    
    "description": "Time of expiration of the challenge in unixtime",
    "validate": "required"
  },
  "ttl": {
    "type": "int",    
    "description": "Time to live in seconds for the challenge",
    "validate": "required"
  },
  "redirect_to": {
    "type": "string",
    "description": "The redirect uri returned upon successful challenge verification",
    "validate": "required, url"
  },
  "code_type": {
    "type": "string",
    "description": "An identifier for the type of code challenge"
  },
  "code": {
    "type": "string",
    "description": "The hashed challenge code. Please do not store plain text codes!",
    "validate": "optional"
  },
  "data": {
    "type": "string",
    "description": "Registered data to the challenge. Can be used to define the data to be executed upon successful challenge"
  },
  "verified_at": {
    "type": "int64",
    "description": "Time of success verification of the challenge in unixtime"    
  }
}
```

### POST /challenges

Create a challenge. Requires scope `idp:create:challenges`.

#### Input
```json
{
  "confirmation_type": {
    "type": "int",
    "description": "Type of challenge. Used to decide what communication to prompt the Identity",
    "validate": "numeric"
  },
  "sub": {
    "type": "string",
    "required": true,
    "description": "The subject for which this challenge is requested. See OAuth2 for details",
    "validate": "uuid",    
  },
  "aud": {
    "type": "string",
    "required": true,    
    "description": "Intended audience for the challenge. See OAuth2 for details"
  },  
  "ttl": {
    "type": "int",    
    "required": true,
    "description": "Time to live in seconds for the challenge"    
  },
  "redirect_to": {
    "type": "string",
    "required": true,
    "description": "The redirect uri returned upon successful challenge verification",
    "validate": "url"
  },
  "code_type": {
    "type": "string",
    "required": true,
    "description": "An identifier for the type of code challenge"    
  },
  "code": {
    "type": "string",
    "required": true,
    "description": "The hashed challenge code. Please do not store plain text codes!"    
  },
  "email": {
    "type": "string",
    "required": true,
    "description": "Email used to send the challenge",
    "validate": "email"
  }  
}
```

#### Output
```json
{
  "otp_challenge": {
    "type": "string",
    "description": "The unique identifier for the challenge",
    "validate": "required, uuid"
  },
  "confirmation_type": {
    "type": "int",
    "description": "Type of challenge. Used to decide what communication to prompt the Identity",
    "validate": "numeric"
  },
  "sub": {
    "type": "string",
    "description": "The subject for which this challenge is requested. See OAuth2 for details",
    "validate": "required, uuid"
  },
  "aud": {
    "type": "string",    
    "description": "Intended audience for the challenge. See OAuth2 for details",
    "validate": "required"
  },
  "iat": {
    "type": "int64",    
    "description": "Time of creation for the challenge in unixtime",
    "validate": "required"
  },
  "exp": {
    "type": "int64",    
    "description": "Time of expiration of the challenge in unixtime",
    "validate": "required"
  },
  "ttl": {
    "type": "int",    
    "description": "Time to live in seconds for the challenge",
    "validate": "required"
  },
  "redirect_to": {
    "type": "string",
    "description": "The redirect uri returned upon successful challenge verification",
    "validate": "required, url"
  },
  "code_type": {
    "type": "string",
    "description": "An identifier for the type of code challenge"
  },
  "code": {
    "type": "string",
    "description": "The hashed challenge code. Please do not store plain text codes!",
    "validate": "optional"
  },
  "data": {
    "type": "string",
    "description": "Registered data to the challenge. Can be used to define the data to be executed upon successful challenge"
  },
  "verified_at": {
    "type": "int64",
    "description": "Time of success verification of the challenge in unixtime"    
  }
}
```

### POST /challenges/verify

Verify a challenge. Requires scope `idp:update:challenges:verify`.

OtpChallenge string `json:"otp_challenge" validate:"required"`
Verified     bool   `json:"verified"      `
RedirectTo   string `json:"redirect_to"   validate:"required,url"`

#### Input
```json
{
  "otp_challenge": {
    "type": "string",
    "required": true
  },
  "code"  : {
    "type": "string",
    "required": true
  }
}
```

#### Output
```json
{
  "otp_challenge": {
    "type": "string",
    "required": true
  },
  "verified"  : {
    "type": "bool",
    "required": true
  },
  "redirect_to": {
    "type": "string",
    "required": true
  }
}
```

## Scopes

The following scopes are required for the endpoints.

| Endpoint                                                                    | Scope                   |
| --------------------------------------------------------------------------- | ----------------------- |
| [POST /identities](#post-identities)                                        | `authenticate:identity` |
| [GET /identities](#get-identities)                                          | `read:identity`         |
| [PUT /identities](#put-identities)                                          | `update:identity`       |
| [DELETE /identities](#delete-identities)                                    | `delete:identity`       |
| [POST /identities/deleteverification](#post-identitiesdeleteverification)   | `delete:identity`       |
| [POST /identities/authenticate](#post-identitiesauthenticate)               | `authenticate:identity` |
| [POST /identities/password](#post-identitiespassword)                       | `authenticate:identity` |
| [POST /identities/recover](#post-identitiesrecover)                         | `recover:identity`      |
| [POST /identities/recoververification](#post-identitiesrecoververification) | `authenticate:identity` |
| [POST /identities/totp](#post-identitiestotptotp)                            | `authenticate:identity` |
| [POST /identities/logout](#post-identitieslogout)                           | `logout:identity`       |

### A note on scopes

The scope `authenticate:identity` is used whenever the password credentials of an Identity is involved. This also include verification codes that are a form of two-factor alias for passwords. This scope should be restricted to applications inside the trust zone only.

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

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idp.identities.post` to call the endpoint or the request will be denied.

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

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idp.authenticate` to call the endpoint or the request will be denied.

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"id":"test", "password":"anewsecret"}'
     -X POST https://id.domain.com/api/identities/password
```

## Authenticate an Identity
To authenticate an Identity, also known as performing a login/signin a `POST` request must be made to the `/identities/authenticate` endpoint. A challenge is required to perform a login. The challenge is obtained by asking Hydra for it when starting the OAuth2 Authorization code flow.

To use the endpoint a client in terms of the OAuth2 protocol is needed. This client needs to have been granted the scope `idp.authenticate` to call the endpoint or the request will be denied.

### Example
```bash
curl -H "Authorization: Bearer <token>" \
     -H "Accept: application/json" -H "Content-Type: application/json" \
     -d '{"challenge":"QWRGvqe56wyega5", "id": "test, ""password":"secret"}'
     -X POST https://id.domain.com/api/identities/authenticate
```
