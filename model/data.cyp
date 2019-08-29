// Root (Master system identity)
MERGE (:Identity {sub:"root", email:"root@localhost",  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"Root", require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})

// Apps (client_id) (pass: 123), should probably be the client_secret
MERGE (:Identity {sub:"idpui", email:"idpui@localhost",  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"IdP UI",  require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
MERGE (:Identity {sub:"idpapi", email:"idpapi@localhost", password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"IdP API", require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
MERGE (:Identity {sub:"aapui", email:"aapui@localhost",  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"AaP UI",  require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
MERGE (:Identity {sub:"aapapi", email:"aapapi@localhost", password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"AaP API", require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
MERGE (:Identity {sub:"hydra", email:"hydra@localhost",  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"Hydra",   require_2fa:false, secret_2fa:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
;
