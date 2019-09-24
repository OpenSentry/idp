package idp

import (
  "crypto/aes"
  "crypto/cipher"
  "crypto/rand"
  "encoding/base64"
  "errors"
  "io"
  "net"
  "net/mail"
  "net/smtp"
  "strings"
  "crypto/tls"
  "fmt"
  "time"
  "golang.org/x/crypto/bcrypt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "github.com/pquerna/otp/totp"
)

type Identity struct {
  Id                   string
  Subject              string
  Name                 string
  Email                string
  Password             string
  AllowLogin           bool
  TotpRequired         bool
  TotpSecret           string
  OtpRecoverCode       string
  OtpRecoverCodeExpire int64
  OtpDeleteCode        string
  OtpDeleteCodeExpire  int64
}

type Challenge struct {
  OtpChallenge string
  Subject      string
  Audience     string
  IssuedAt     int64
  ExpiresAt    int64
  TTL          int64
  RedirectTo   string
  CodeType     string
  Code         string
  Verified     int64
}

type Invite struct {
  Id string
  Email string
  Username string
  GrantedScopes string
  FollowIdentities string
  ExpiresInSeconds int64
  IssuedAt int64
  ExpiresAt int64
  InviterIdentityId string
  InvitedIdentityId string
}

type Follow struct {
  Id string
  FollowIdentity string
}

type IdentityInvite struct {
  Id string
  TTL int64
  IssuedAt int64
  ExpiresAt int64
  InvitedBy string
  Email string
  InvitedIdentityId string
  Username string
}

type RecoverChallenge struct {
  Id               string
  VerificationCode string
  Expire           int64
  RedirectTo       string
}

type DeleteChallenge struct {
  Id               string
  VerificationCode string
  Expire           int64
  RedirectTo       string
}

func marshalRecordToIdentity(record neo4j.Record) (Identity) {
  // NOTE: This means the statment sequence of the RETURN (possible order by)
  // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
  // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
  // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
  id                   := record.GetByIndex(0).(string)
  sub                  := record.GetByIndex(1).(string)
  password             := record.GetByIndex(2).(string)
  name                 := record.GetByIndex(3).(string)
  email                := record.GetByIndex(4).(string)
  allowLogin           := record.GetByIndex(5).(bool)
  totpRequired         := record.GetByIndex(6).(bool)
  totpSecret           := record.GetByIndex(7).(string)
  otpRecoverCode       := record.GetByIndex(8).(string)
  otpRecoverCodeExpire := record.GetByIndex(9).(int64)
  otpDeleteCode        := record.GetByIndex(10).(string)
  otpDeleteCodeExpire  := record.GetByIndex(11).(int64)

  return Identity{
    Id: id,
    Subject: sub,
    Name: name,
    Email: email,
    AllowLogin: allowLogin,
    Password: password,
    TotpRequired: totpRequired,
    TotpSecret: totpSecret,
    OtpRecoverCode: otpRecoverCode,
    OtpRecoverCodeExpire: otpRecoverCodeExpire,
    OtpDeleteCode: otpDeleteCode,
    OtpDeleteCodeExpire: otpDeleteCodeExpire,
  }
}

func marshalRecordToChallenge(record neo4j.Record) (Challenge) {
  otpChallenge := record.GetByIndex(0).(string)
  aud          := record.GetByIndex(1).(string)
  iat          := record.GetByIndex(2).(int64)
  exp          := record.GetByIndex(3).(int64)
  verified     := record.GetByIndex(4).(int64)
  ttl          := record.GetByIndex(5).(int64)
  codeType     := record.GetByIndex(6).(string)
  code         := record.GetByIndex(7).(string)
  redirectTo   := record.GetByIndex(8).(string)
  sub          := record.GetByIndex(9).(string)

  return Challenge{
    OtpChallenge: otpChallenge,
    Subject: sub,
    Audience: aud,
    IssuedAt: iat,
    ExpiresAt: exp,
    Verified: verified,
    TTL: ttl,
    RedirectTo: redirectTo,
    CodeType: codeType,
    Code: code,
  }
}

func marshalRecordToIdentityInvite(record neo4j.Record) (IdentityInvite) {
  id               := record.GetByIndex(0).(string)
  email            := record.GetByIndex(1).(string)
  username         := record.GetByIndex(2).(string)
  ttl              := record.GetByIndex(3).(int64)
  iat              := record.GetByIndex(4).(int64)
  exp              := record.GetByIndex(5).(int64)
  invitedBy        := record.GetByIndex(6).(string)

  return IdentityInvite{
    Id: id,
    Email: email,
    Username: username,
    TTL: ttl,
    IssuedAt: iat,
    ExpiresAt: exp,
    InvitedBy: invitedBy,
  }
}

func marshalRecordToFollow(record neo4j.Record) (Follow) {
  from := record.GetByIndex(0).(string)
  to   := record.GetByIndex(1).(string)

  return Follow{
    Id: from,
    FollowIdentity: to,
  }
}

func marshalRecordToInvite(record neo4j.Record) (Invite) {
  id               := record.GetByIndex(0).(string)
  email            := record.GetByIndex(1).(string)
  username         := record.GetByIndex(2).(string)
  grantedScopes    := record.GetByIndex(3).(string)
  followIdentities := record.GetByIndex(4).(string)
  iat              := record.GetByIndex(5).(int64)
  ttl              := record.GetByIndex(6).(int64)
  exp              := record.GetByIndex(7).(int64)
  inviterId        := record.GetByIndex(8).(string)

  var invitedId string = ""
  __invitedId        := record.GetByIndex(9)
  if __invitedId != nil {
    invitedId = __invitedId.(string)
  }

  return Invite{
    Id: id,
    Email: email,
    Username: username,
    GrantedScopes: grantedScopes,
    FollowIdentities: followIdentities,
    ExpiresInSeconds: ttl,
    IssuedAt: iat,
    ExpiresAt: exp,
    InviterIdentityId: inviterId,
    InvitedIdentityId: invitedId,
  }
}

func FetchInvitesForIdentity(driver neo4j.Driver, identity Identity) ([]Invite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (i:Identity {id:$id})
      MATCH (inv:Invite)-[:INVITED_BY]->(i) WHERE inv.exp > datetime().epochSeconds

      WITH i, inv

      OPTIONAL MATCH (n:Identity)-[:IS_INVITED]->(inv)-[:INVITED_BY]->(i)

      WITH i, inv, n

      RETURN inv.id, inv.email, inv.username, inv.granted_scopes, inv.follow_identities, inv.iat, inv.ttl, inv.exp, i.id, n.id
    `
    params := map[string]interface{}{"id": identity.Id}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret []Invite
    for result.Next() {
      record := result.Record()
      inv := marshalRecordToInvite(record)
      ret = append(ret, inv)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return nil, err
  }
  if obj != nil {
    return obj.([]Invite), nil
  }
  return nil, nil
}

func FetchInviteById(driver neo4j.Driver, id string) (Invite, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Invite{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (inv:Invite {id:$id})-[:INVITED_BY]->(i) WHERE inv.exp > datetime().epochSeconds

      WITH i, inv

      OPTIONAL MATCH (n:Identity)-[:IS_INVITED]->(inv)-[:INVITED_BY]->(i)

      WITH i, inv, n

      RETURN inv.id, inv.email, inv.username, inv.granted_scopes, inv.follow_identities, inv.iat, inv.ttl, inv.exp, i.id, n.id
    `
    params := map[string]interface{}{"id": id}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Invite
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToInvite(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return Invite{}, false, err
  }
  if obj != nil {
    return obj.(Invite), true, nil
  }
  return Invite{}, false, nil
}

func AcceptInvite(driver neo4j.Driver, invite Invite) (Invite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (inv:Invite {id:$id})

      // Granted scopes

      // Follows

      WITH i, inv

      OPTIONAL MATCH (n:Identity {email:inv.email})

      WITH i, inv, n, collect(n) as c

      FOREACH( n in c | MERGE (n)-[:IS_INVITED]->(inv) )

      WITH i, inv, n

      OPTIONAL MATCH (i)<-[:INVITED_BY]-(d:Invite) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN inv.id, inv.email, inv.username, inv.granted_scopes, inv.follow_identities, inv.iat, inv.ttl, inv.exp, i.id, n.id
    `
    params := map[string]interface{}{
      "id": invite.Id,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Invite
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToInvite(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Invite{}, err
  }
  return obj.(Invite), nil
}

func CreateFollow(driver neo4j.Driver, from Identity, to Identity) (Follow, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Follow{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (from:Identity {id:$from})
      MATCH (to:Identity {id:$to})
      MERGE (from)-[:FOLLOW]->(to)
      RETURN from.id, to.id
    `
    params := map[string]interface{}{
      "from": from.Id,
      "to": to.Id,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Follow
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToFollow(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Follow{}, err
  }
  return obj.(Follow), nil
}


func CreateIdentityInvite(driver neo4j.Driver, invite IdentityInvite) (IdentityInvite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return IdentityInvite{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {id:$id})
      MERGE (i)<-[:INVITED_BY]-(inv:Identity:Invite {id:randomUUID(), iat:datetime().epochSeconds, exp:datetime().epochSeconds + $ttl, ttl:$ttl})
      MERGE (inv)-[:HINT]->(u:Username {username:$username})

      WITH i, inv, u

      OPTIONAL MATCH (invited:Identity {email:$email})

      WITH i, inv, u, collect(invited) as invited

      FOREACH( n in invited | MERGE (n)-[:IS_INVITED]->(inv) )

      WITH i, inv, u

      MERGE (inv)-[:SENT_TO]->(e:Email {email:$email})

      RETURN inv.id, e.email, u.username, inv.ttl, inv.iat, inv.exp, i.id
    `
    params := map[string]interface{}{
      "email": invite.Email,
      "username": invite.Username,
      "ttl": invite.TTL,
      "id": invite.InvitedBy,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret IdentityInvite
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentityInvite(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return IdentityInvite{}, err
  }
  return obj.(IdentityInvite), nil
}

func FetchIdentityInviteById(driver neo4j.Driver, id string) (IdentityInvite, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return IdentityInvite{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (inv:IdentityInvite {id:$id})-[:INVITED_BY]->(i) WHERE inv.exp > datetime().epochSeconds
      MATCH (inv)-[:SENT_TO]->(e:Email)
      MATCH (inv)-[:HINT]->(u:Username)
      RETURN inv.id, e.email, u.username, inv.ttl, inv.iat, inv.exp, i.id
    `
    params := map[string]interface{}{"id": id}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret IdentityInvite
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentityInvite(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return IdentityInvite{}, false, err
  }
  if obj != nil {
    return obj.(IdentityInvite), true, nil
  }
  return IdentityInvite{}, false, nil
}

func CreateInvite(driver neo4j.Driver, identity Identity, invite Invite) (Invite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {id:$id})
      CREATE (i)<-[:INVITED_BY]-(inv:Invite {id:randomUUID(), email:$email, username:$username, granted_scopes:$grantedScopes, follow_identities:$followIdentities, iat:datetime().epochSeconds, exp:datetime().epochSeconds + $ttl, ttl:$ttl})

      WITH i, inv

      OPTIONAL MATCH (n:Identity {email:inv.email})

      WITH i, inv, n, collect(n) as c

      FOREACH( n in c | MERGE (n)-[:IS_INVITED]->(inv) )

      WITH i, inv, n

      OPTIONAL MATCH (i)<-[:INVITED_BY]-(d:Invite) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN inv.id, inv.email, inv.username, inv.granted_scopes, inv.follow_identities, inv.iat, inv.ttl, inv.exp, i.id, n.id
    `
    params := map[string]interface{}{
      "id": identity.Id,
      "email": invite.Email,
      "username": invite.Username,
      "grantedScopes": invite.GrantedScopes,
      "followIdentities": invite.FollowIdentities,
      "ttl": invite.ExpiresInSeconds,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Invite
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToInvite(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Invite{}, err
  }
  return obj.(Invite), nil
}

func ValidatePassword(storedPassword string, password string) (bool, error) {
  err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
  if err != nil {
		return false, err
	}
  return true, nil
}

func CreatePassword(password string) (string, error) {
  hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
  if err != nil {
    return "", err
  }
  return string(hash), nil
}

func ValidateOtp(otp string, secret string) (bool, error) {
  valid := totp.Validate(otp, secret)
  return valid, nil
}

func CreateDeleteChallenge(url string, identity Identity, challengeTimeoutInSeconds int64) (DeleteChallenge, error) {
  verificationCode, err := GenerateRandomDigits(6);
  if err != nil {
    return DeleteChallenge{}, err
  }

  timeout := time.Duration(challengeTimeoutInSeconds)
  expirationTime := time.Now().Add(timeout * time.Second)
  expiresAt := expirationTime.Unix()
  redirectTo := url

  return DeleteChallenge{
    Id: identity.Id,
    VerificationCode: verificationCode,
    Expire: expiresAt,
    RedirectTo: redirectTo,
  }, nil
}

func CreateRecoverChallenge(url string, identity Identity, challengeTimeoutInSeconds int64) (RecoverChallenge, error) {
  verificationCode, err := GenerateRandomDigits(6);
  if err != nil {
    return RecoverChallenge{}, err
  }

  timeout := time.Duration(challengeTimeoutInSeconds)
  expirationTime := time.Now().Add(timeout * time.Second)
  expiresAt := expirationTime.Unix()
  redirectTo := url

  return RecoverChallenge{
    Id: identity.Id,
    VerificationCode: verificationCode,
    Expire: expiresAt,
    RedirectTo: redirectTo,
  }, nil
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func GenerateRandomDigits(max int) (string, error) {
  b := make([]byte, max)
  n, err := io.ReadAtLeast(rand.Reader, b, max)
  if n != max {
    return "", err
  }
  for i := 0; i < len(b); i++ {
    b[i] = table[int(b[i])%len(table)]
  }
  return string(b), nil
}

// Enforce AES-256 by using 32 byte string as key param
func Encrypt(str string, key string) (string, error) {

   bKey, err := base64.StdEncoding.DecodeString(key)
   if err != nil {
     return "", err
   }

   bStr := []byte(str)
   bEncryptedStr, err := encrypt(bStr, bKey)
   if err != nil {
     return "", err
   }

   return base64.StdEncoding.EncodeToString(bEncryptedStr), nil
}

// Enforce AES-256 by using 32 byte string as key param
func Decrypt(str string, key string) (string, error) {

  bKey, err := base64.StdEncoding.DecodeString(key)
  if err != nil {
    return "", err
  }

  bStr, err := base64.StdEncoding.DecodeString(str)
  if err != nil {
    return "", err
  }

  bDecryptedStr, err := decrypt(bStr, bKey)
  if err != nil {
    return "", err
  }
  return string(bDecryptedStr), nil
}

// The key argument should be 32 bytes to use AES-256
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
  c, err := aes.NewCipher(key)
  if err != nil {
    return nil, err
  }

  gcm, err := cipher.NewGCM(c)
  if err != nil {
    return nil, err
  }

  nonce := make([]byte, gcm.NonceSize())
  if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
    return nil, err
  }

  return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// The key argument should be 32 bytes to use AES-256
func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
  c, err := aes.NewCipher(key)
  if err != nil {
    return nil, err
  }

  gcm, err := cipher.NewGCM(c)
  if err != nil {
    return nil, err
  }

  nonceSize := gcm.NonceSize()
  if len(ciphertext) < nonceSize {
    return nil, errors.New("ciphertext too short")
  }

  nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
  return gcm.Open(nil, nonce, ciphertext, nil)
}

func FetchChallenge(driver neo4j.Driver, challenge string) (Challenge, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (c:Challenge {otp_challenge:$challenge})<-[:REQUESTED]-(i:Human:Identity) WHERE c.exp > datetime().epochSeconds
      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.id
    `
    params := map[string]interface{}{"challenge": challenge}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Challenge
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToChallenge(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return Challenge{}, false, err
  }
  if obj != nil {
    return obj.(Challenge), true, nil
  }
  return Challenge{}, false, nil
}

func VerifyChallenge(driver neo4j.Driver, challenge Challenge) (Challenge, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (c:Challenge {otp_challenge:$challenge})<-[:REQUESTED]-(i:Human:Identity) SET c.verified = datetime().epochSeconds
      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.id
    `
    params := map[string]interface{}{"challenge": challenge.OtpChallenge}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Challenge
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToChallenge(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return Challenge{}, false, err
  }
  if obj != nil {
    return obj.(Challenge), true, nil
  }
  return Challenge{}, false, nil
}



func CreateChallengeForIdentity(driver neo4j.Driver, identity Identity, challenge Challenge) (Challenge, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {id:$id})
      MERGE (i)-[:REQUESTED]->(c:Challenge {otp_challenge:randomUUID(), aud:$aud, iat:datetime().epochSeconds, exp:datetime().epochSeconds + $ttl, verified:0, ttl:$ttl, code_type:$codeType, code:$code, redirect_to:$redirectTo})

      WITH i, c

      OPTIONAL MATCH (i)-[:REQUESTED]->(d:Challenge) WHERE id(c) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.id
    `
    params := map[string]interface{}{
      "id": identity.Id,
      "aud": challenge.Audience,
      "ttl": challenge.TTL,
      "codeType": challenge.CodeType,
      "code": challenge.Code,
      "redirectTo": challenge.RedirectTo,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Challenge
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToChallenge(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return ret, nil
  })

  if err != nil {
    return Challenge{}, false, err
  }
  if obj != nil {
    return obj.(Challenge), true, nil
  }
  return Challenge{}, false, nil
}

func UpdateAllowLogin(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.allow_login=$allowLogin
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "allowLogin": identity.AllowLogin}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func UpdateTotp(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.totp_required=$required, i.totp_secret=$secret
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "required": identity.TotpRequired, "secret": identity.TotpSecret}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func UpdateOtpDeleteCode(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.otp_delete_code=$code, i.otp_delete_code_expire=$expire
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "code": identity.OtpDeleteCode, "expire": identity.OtpDeleteCodeExpire}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func UpdateOtpRecoverCode(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.otp_recover_code=$code, i.otp_recover_code_expire=$expire
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "code": identity.OtpRecoverCode, "expire": identity.OtpRecoverCodeExpire}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func UpdatePassword(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.password=$password
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "password": identity.Password}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func CreateIdentity(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      CREATE (i:Human:Identity {id:randomUUID(), sub:$sub, password:$password, name:$name, email:$email, allow_login:true, totp_required:false, totp_secret:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Subject, "password": identity.Password, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

// NOTE: This can update everything that is _NOT_ sensitive to the authentication process like Identity.Password
//       To change the password see recover for that.
func UpdateIdentity(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) WITH i SET i.name=$name, i.email=$email
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": identity.Id, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()
      ret = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func DeleteIdentity(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) DETACH DELETE i
    `
    params := map[string]interface{}{"id": identity.Id}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    result.Next()

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return Identity{}, err
    }
    return Identity{Id: identity.Id}, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return obj.(Identity), nil
}

func FetchIdentityById(driver neo4j.Driver, id string) (Identity, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Identity{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (i:Human:Identity {id: $id})
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"id": id}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var identity Identity
    if result.Next() {
      record := result.Record()
      identity = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return Identity{}, err
    }
    return identity, nil
  })

  if err != nil {
    return Identity{}, false, err
  }
  if obj != nil {
    return obj.(Identity), true, nil
  }
  return Identity{}, false, nil
}

func FetchIdentityByEmail(driver neo4j.Driver, email string) (Identity, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Identity{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (i:Human:Identity {email: $email})
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"email": email}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var identity Identity
    if result.Next() {
      record := result.Record()
      identity = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return Identity{}, err
    }
    return identity, nil
  })

  if err != nil {
    return Identity{}, false, err
  }
  if obj != nil {
    return obj.(Identity), true, nil
  }
  return Identity{}, false, nil
}

func FetchIdentityBySubject(driver neo4j.Driver, sub string) (Identity, bool, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Identity{}, false, err
  }
  defer session.Close()

  obj, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (i:Human:Identity {sub: $sub})
      RETURN i.id, i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": sub}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var identity Identity
    if result.Next() {
      record := result.Record()
      identity = marshalRecordToIdentity(record)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return Identity{}, err
    }
    return identity, nil
  })

  if err != nil {
    return Identity{}, false, err
  }
  if obj != nil {
    return obj.(Identity), true, nil
  }
  return Identity{}, false, nil
}

// EMAIL BEGIN

type SMTPSender struct {
  Name string
  Email string
  ReturnPath string
}

type SMTPConfig struct {
  Host string
  Username string
  Password string
  Sender SMTPSender
  SkipTlsVerify int
}

type AnEmail struct {
  Subject string
  Body string
}

func encodeRFC2047(String string) string {
	// use mail's rfc2047 to encode any string
	addr := mail.Address{String, ""}
	return strings.Trim(addr.String(), " <>")
}

type unencryptedAuth struct {
    smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
    s := *server
    s.TLS = true
    return a.Auth.Start(&s)
}

func SendAnEmailToAnonymous(smtpConfig SMTPConfig, name string, email string, anEmail AnEmail) (bool, error) {
  return sendAnEmail(smtpConfig, name, email, anEmail)
}

func SendAnEmailToIdentity(smtpConfig SMTPConfig, identity Identity, anEmail AnEmail) (bool, error) {
  return sendAnEmail(smtpConfig, identity.Name, identity.Email, anEmail)
}

func sendAnEmail(smtpConfig SMTPConfig, name string, email string, anEmail AnEmail) (bool, error) {

  from := mail.Address{smtpConfig.Sender.Name, smtpConfig.Sender.Email}
  to := mail.Address{name, email}

  subject := anEmail.Subject
  body := anEmail.Body

  header := make(map[string]string)
  header["Return-Path"] = smtpConfig.Sender.ReturnPath
  header["From"] = from.String()
  header["To"] = to.String()
  header["Subject"] = encodeRFC2047(subject)
  header["MIME-Version"] = "1.0"
  header["Content-Type"] = "text/plain; charset=\"utf-8\""
  header["Content-Transfer-Encoding"] = "base64"

  message := ""
  for k, v := range header {
    message += fmt.Sprintf("%s: %s\r\n", k, v)
  }
  message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

  host, _, _ := net.SplitHostPort(smtpConfig.Host)

  // Trick go library into thinking we are encrypting password to allow SMTP with authentication but no encryption
  //auth := unencryptedAuth { smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, host) }
  auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, host)

  /*err := smtp.SendMail(smtpConfig.Host, auth, smtpConfig.Sender.Email, []string{identity.Email}, []byte(message))
  if err != nil {
    return false, err
  }
  return true, nil*/

  tlsconfig := &tls.Config {
    InsecureSkipVerify: smtpConfig.SkipTlsVerify == 1, // Using selfsigned certs
    ServerName: host,
  }

  // Here is the key, you need to call tls.Dial instead of smtp.Dial
  // for smtp servers running on 465 that require an ssl connection
  // from the very beginning (no starttls)
  /*conn, err := tls.Dial("tcp", smtpConfig.Host, tlsconfig)
  if err != nil {
    return false, err
  }

  c, err := smtp.NewClient(conn, host)
  if err != nil {
    return false, err
  }
  */

  c, err := smtp.Dial(smtpConfig.Host)
  if err != nil {
    return false, err
  }

  err = c.StartTLS(tlsconfig)

  // Auth
  if err := c.Auth(auth); err != nil {
    return false, err
  }

  // To && From
  if err = c.Mail(from.Address); err != nil {
    return false, err
  }

  if err = c.Rcpt(to.Address); err != nil {
    return false, err
  }

  // Data
  w, err := c.Data()
  if err != nil {
    return false, err
  }

  _, err = w.Write([]byte(message))
  if err != nil {
    return false, err
  }

  err = w.Close()
  if err != nil {
    return false, err
  }

  c.Quit()
  return true, nil
}
