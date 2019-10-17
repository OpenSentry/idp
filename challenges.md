# Challenges

To verify an identity or verify access to a resource.

## Challenge Types

### Timed One Time Password

Using a shared secret and time. Calculate a password accounting for clock skew and drift. Often using an Authenticator App.


### One Time Password

Using a randomly generated code stored in DB. Assert identity or resource ownership. Delivered to the identity or resource.




# Email Confirmation

Verify access to an email resource. Send an OTP to the email.




# Use Cases


## Identity Creation
When registering an Identity with IDP. We need to confirm control over a recovery method before creating the Identity.

To create an Identity for an Email -> IDP sends an OTP to the Email -> Upon verification of OTP -> Register the Identity.



## Identity Authentication

To authenticate an Identity -> Redirect Identity to IDP login -> Upon verification of password -> Check for 2fa -> Upon verification of TOTP or OTP -> Identity Authenticated



## Identity Email Change

To change the Email of an Identity -> Check for 2fa -> Upon verification of TOTP or OTP -> IDP sends an OTP to the new Email -> Upon verification of OTP -> Change Email



## Identity Password Change

To change the password of an Identity -> Check for 2fa -> Upon verification of TOTP or OTP -> Change Password



## Identity Recovery

To recover an Identity -> IDP send OTP to recovery method -> Upon verification of TOTP or OTP -> Start Identity Password Change



## Identity Deletion

To delete an Identity -> Check for 2fa -> Upon verification of TOTP or OTP -> IDP send OTP to recovery method ->  Upon verification of TOTP or OTP -> Delete Identity