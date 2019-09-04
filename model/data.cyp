// Root (Master system identity)
MERGE (:Identity {sub:"root", email:"root@localhost",  password:"$2a$10$SOyUCy0KLFQJa3xN90UgMe9q5wE.LfakmkCsfKLCIjRY6.CcRDYwu", name:"Root", totp_required:false, totp_secret:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
