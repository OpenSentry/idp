// Root (Master system identity)
MERGE (:Identity:Human {
  id:randomUUID(),
  username:"root",
  email:"root@localhost",
  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu",
  allow_login:true,
  name:"Root",
  totp_required:false,
  totp_secret:"",
  otp_recover_code:"",
  otp_recover_code_expire:0,
  otp_delete_code:"",
  otp_delete_code_expire:0
})

// ### Required clients
MERGE (:Identity:Client {
  id:randomUUID(),
  client_id:"idp",
  client_secret:"",
  name: "IDP hydra client",
  description:"Used by the Identity Provider api to call Hydra"
})

MERGE (:Identity:Client {
  id:randomUUID(),
  client_id:"idpui",
  client_secret:"",
  name: "IDP api client",
  description:"Used by the Identity Provider UI to call the Identity Provider API"
});

// ## IDPAPI
MERGE (:Identity:ResourceServer {
  id:randomUUID(),
  name:"IDP",
  aud:"idp",
  description:"Identity Provider"
});

// HYDRA API
MERGE (:Identity:ResourceServer {
  id:randomUUID(),
  name:"Hydra",
  description:"OAuth2 API"
})
;

// # AAP
MERGE (:Identity:Client {
  id:randomUUID(),
  client_id:"aap",
  client_secret:"",
  name: "AAP hydra client",
  description:"Used by the Access and Authorization Provider api to call Hydra
"})
MERGE (:Identity:Client {
  id:randomUUID(),
  client_id:"aapui",
  client_secret:"",
  name: "AAP api client",
  description:"Used by the Access and Authorization Provider UI to call the Access and Authorization API"
})
;

// AAP API
MERGE (:Identity:ResourceServer {
  id:randomUUID(),
  name:"AAP",
  description:"Access and Authorization provider"
})
;
