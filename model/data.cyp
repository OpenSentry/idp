// # Resource servers

// ## IDP
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"IDP",
  aud:"idp",
  description:"Identity Provider"
})
;

// ## AAP
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"AAP",
  aud:"aap",
  description:"Access and Authorization provider"
})
;

// ## HYDRA
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"Hydra",
  aud:"hydra",
  description:"OAuth2 API"
})
;


// # Clients

// ## IDP -> Hydra
MATCH (rs:Identity:ResourceServer {aud: "hydra"})
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP hydra client",
  client_id:"idp",
  client_secret:"",
  description:"Used by the Identity Provider api to call Hydra",
  grant_types: [
    "client_credentials"
  ],
  redirect_uris: [],
  response_types: [
    "token"
  ],
  token_endpoint_auth_method: "client_secret_basic"
})-[:AUDIENCE]->(rs)
;

// ## AAP -> Hydra
MATCH (rs:Identity:ResourceServer {aud: "hydra"})
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "AAP hydra client",
  client_id:"aap",
  client_secret:"",
  description:"Used by the Access and Authorization Provider api to call Hydra",
  grant_types: [
    "client_credentials"
  ],
  redirect_uris: [],
  response_types: [
    "token"
  ],
  token_endpoint_auth_method: "client_secret_basic"
})-[:AUDIENCE]->(rs)
;

// ## IDPUI -> IDP
MATCH (rs:Identity:ResourceServer {aud: "idp"})
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP api client",
  client_id:"idpui",
  client_secret:"",
  description:"Used by the Identity Provider UI to call the Identity Provider API",
  grant_types: [
    "authorization_code",
    "client_credentials",
    "refresh_token"
  ],
  redirect_uris: [
    "https://id.localhost/callback"
  ],
  response_types: [
    "token",
    "code"
  ],
  token_endpoint_auth_method: "client_secret_basic"
})-[:AUDIENCE]->(rs)
;


// ## AAPUI -> AAP
MATCH (rs:Identity:ResourceServer {aud: "aap"})
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "AAP api client",
  client_id:"aapui",
  client_secret:"",
  description:"Used by the Access and Authorization Provider UI to call the Access and Authorization API",
  grant_types: [
    "authorization_code",
    "client_credentials",
    "refresh_token"
  ],
  redirect_uris: [
    "https://aa.localhost/callback"
  ],
  response_types: [
    "token",
    "code"
  ],
  token_endpoint_auth_method: "client_secret_basic"
})-[:AUDIENCE]->(rs)
;



// ## MEUI -> IDP & AAP
// this one doesnt really belong here, does it?
MERGE (c:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP & AAP client for MEUI",
  client_id:"meui",
  client_secret:"",
  description:"Used by the Me UI to call the Identity Provider & Access Provider API",
  grant_types: [
    "authorization_code",
    "refresh_token"
  ],
  redirect_uris: [
    "https://me.localhost/callback"
  ],
  response_types: [
    "token",
    "code"
  ],
  token_endpoint_auth_method: "client_secret_basic"
})

WITH c

MATCH (rs:Identity:ResourceServer)
WHERE rs.aud IN ["aap", "idp"]
MERGE (c)-[:AUDIENCE]->(rs)
;
