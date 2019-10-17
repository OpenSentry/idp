// ### Required clients
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  client_id:"idp",
  client_secret:"",
  name: "IDP hydra client",
  description:"Used by the Identity Provider api to call Hydra"
})

MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  client_id:"idpui",
  client_secret:"",
  name: "IDP api client",
  description:"Used by the Identity Provider UI to call the Identity Provider API"
});


// this one doesnt really belong here, does it?
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  client_id:"meui",
  client_secret:"",
  name: "IDP & AAP client for MEUI",
  description:"Used by the Me UI to call the Identity Provider & Access Provider API"
});


// ## IDPAPI
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"IDP",
  aud:"idp",
  description:"Identity Provider"
});

// HYDRA API
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"Hydra",
  aud:"hydra",
  description:"OAuth2 API"
})
;

// # AAP
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  client_id:"aap",
  client_secret:"",
  name: "AAP hydra client",
  description:"Used by the Access and Authorization Provider api to call Hydra
"})
MERGE (:Identity:Client {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  client_id:"aapui",
  client_secret:"",
  name: "AAP api client",
  description:"Used by the Access and Authorization Provider UI to call the Access and Authorization API"
})
;

// AAPAPI
MERGE (:Identity:ResourceServer {
  id:randomUUID(), iat:datetime().epochSeconds, iss:"https://id.localhost", exp:0,
  name:"AAP",
  aud:"aap",
  description:"Access and Authorization provider"
})
;
