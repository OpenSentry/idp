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
  id:"8dc7ea3e-c61a-47cd-acf2-2f03615e3f8b", iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP hydra client",
  client_secret:"", // To migrate this it needs to be encrypted before stored.
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
  id:"b27062eb-090a-4c9a-a982-ff47b8c7f916", iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "AAP hydra client",
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
  id:"c7f1afc4-1e1f-484e-b3c2-0519419690cb", iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP api client",
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
  id:"919e2026-06af-4c82-9d84-6af4979d9e7a", iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "AAP api client",
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
  id:"20f2bfc6-44df-424a-b490-c024d009892c", iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name: "IDP & AAP client for MEUI",
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
