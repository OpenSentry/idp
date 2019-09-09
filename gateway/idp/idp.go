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
  Id                   string `json:"id" binding:"required"`
  Name                 string `json:"name"`
  Email                string `json:"email"`
  Password             string `json:"password"`
  AllowLogin           bool   `json:"allow_login"`
  TotpRequired         bool   `json:"totp_required"`
  TotpSecret           string `json:"totp_secret"`
  OtpRecoverCode       string `json:"otp_recover_code"`
  OtpRecoderCodeExpire int64  `json:"otp_recover_code_expire"`
  OtpDeleteCode        string `json:"otp_delete_code"`
  OtpDeleteCodeExpire  int64  `json:"otp_delete_code_expire"`
}

type Challenge struct {
  OtpChallenge string `json:"otp_challenge"`
  Subject      string `json:"sub"`
  Audience     string `json:"aud"`
  IssuedAt     int64  `json:"iat"`
  ExpiresAt    int64  `json:"exp"`
  TTL          int64  `json:"ttl"`
  RedirectTo   string `json:"redirect_to"`
  CodeType     string `json:"code_type"`
  Code         string `json:"code"`
  Verified     int64  `json:"verified"`
}

type RecoverChallenge struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Expire           int64  `json:"expire" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type DeleteChallenge struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Expire           int64  `json:"expire" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
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

func FetchChallenge(driver neo4j.Driver, otpChallenge string) (Challenge, bool, error) {
  var err error
  var session neo4j.Session
  var obj interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (c:Challenge {otp_challenge:$otpChallenge})<-[:REQUESTED]-(i:Identity) WHERE c.exp > datetime().epochSeconds
      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.sub
    `
    params := map[string]interface{}{"otpChallenge": otpChallenge}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var challenge Challenge
    if result.Next() {
      record := result.Record()

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

      challenge = Challenge{
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

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return challenge, nil
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
  var session neo4j.Session
  var obj interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (c:Challenge {otp_challenge:$otpChallenge})<-[:REQUESTED]-(i:Identity) SET c.verified = datetime().epochSeconds
      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.sub
    `
    params := map[string]interface{}{"otpChallenge": challenge.OtpChallenge}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var challenge Challenge
    if result.Next() {
      record := result.Record()

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

      challenge = Challenge{
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

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return challenge, nil
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
  var session neo4j.Session
  var obj interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, false, err
  }
  defer session.Close()

  obj, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub})
      CREATE (i)-[:REQUESTED]->(c:Challenge {otp_challenge:randomUUID(), aud:$aud, iat:datetime().epochSeconds, exp:datetime().epochSeconds + $ttl, verified:0, ttl:$ttl, code_type:$codeType, code:$code, redirect_to:$redirectTo})

      WITH i, c

      MATCH (i)-[:REQUESTED]->(d:Challenge) WHERE d.exp <= datetime().epochSeconds DETACH DELETE d

      RETURN c.otp_challenge, c.aud, c.iat, c.exp, c.verified, c.ttl, c.code_type, c.code, c.redirect_to, i.sub
    `
    params := map[string]interface{}{
      "sub": identity.Id,
      "aud": challenge.Audience,
      "ttl": challenge.TTL,
      "codeType": challenge.CodeType,
      "code": challenge.Code,
      "redirectTo": challenge.RedirectTo,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var challenge Challenge
    if result.Next() {
      record := result.Record()

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

      challenge = Challenge{
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

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }

    // TODO: Check if neo returned empty set

    return challenge, nil
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
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) SET i.allow_login=$allowLogin
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "allowLogin": identity.AllowLogin}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      ret = identity
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
  return id.(Identity), nil
}

func UpdateTotp(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) SET i.totp_required=$required, i.totp_secret=$secret
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "required": identity.TotpRequired, "secret": identity.TotpSecret}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      ret = identity
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
  return id.(Identity), nil
}

func UpdateOtpDeleteCode(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) SET i.otp_delete_code=$code, i.otp_delete_code_expire=$expire
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "code": identity.OtpDeleteCode, "expire": identity.OtpDeleteCodeExpire}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      ret = identity
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
  return id.(Identity), nil
}

func UpdateOtpRecoverCode(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) SET i.otp_recover_code=$code, i.otp_recover_code_expire=$expire
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "code": identity.OtpRecoverCode, "expire": identity.OtpRecoderCodeExpire}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      ret = identity
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
  return id.(Identity), nil
}

func UpdatePassword(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) SET i.password=$password
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "password": identity.Password}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      ret = identity
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
  return id.(Identity), nil
}

func CreateIdentities(driver neo4j.Driver, identity Identity) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      CREATE (i:Identity {sub:$sub, password:$password, name:$name, email:$email, allow_login:true, totp_required:false, totp_secret:"", otp_recover_code:"", otp_recover_code_expire:0, otp_delete_code:"", otp_delete_code_expire:0})
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "password": identity.Password, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var identities []Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      identities = append(identities, identity)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return identities, nil
  })

  if err != nil {
    return nil, err
  }
  return ids.([]Identity), nil
}

// NOTE: This can update eveything but the Identity.sub and Identity.password
//       To change the password see recover for that.
func UpdateIdentities(driver neo4j.Driver, identity Identity) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) WITH i SET i.name=$name, i.email=$email
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": identity.Id, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var identities []Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
      identities = append(identities, identity)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return identities, nil
  })

  if err != nil {
    return nil, err
  }
  return ids.([]Identity), nil
}

func DeleteIdentity(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Identity {sub:$sub}) DETACH DELETE i
    `
    params := map[string]interface{}{"sub": identity.Id}
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
  return id.(Identity), nil
}

// https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
func FetchIdentity(driver neo4j.Driver, sub string) (Identity, bool, error) {
  var err error
  var session neo4j.Session
  var obj interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return Identity{}, false, err
  }
  defer session.Close()

  obj, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := `
      MATCH (i:Identity {sub: $sub})
      RETURN i.sub, i.password, i.name, i.email, i.allow_login, i.totp_required, i.totp_secret, i.otp_recover_code, i.otp_recover_code_expire, i.otp_delete_code, i.otp_delete_code_expire
    `
    params := map[string]interface{}{"sub": sub}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var identity Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub                  := record.GetByIndex(0).(string)
      password             := record.GetByIndex(1).(string)
      name                 := record.GetByIndex(2).(string)
      email                := record.GetByIndex(3).(string)
      allowLogin           := record.GetByIndex(4).(bool)
      totpRequired         := record.GetByIndex(5).(bool)
      totpSecret           := record.GetByIndex(6).(string)
      otpRecoverCode       := record.GetByIndex(7).(string)
      otpRecoverCodeExpire := record.GetByIndex(8).(int64)
      otpDeleteCode        := record.GetByIndex(9).(string)
      otpDeleteCodeExpire  := record.GetByIndex(10).(int64)

      identity = Identity{
        Id: sub,
        Name: name,
        Email: email,
        AllowLogin: allowLogin,
        Password: password,
        TotpRequired: totpRequired,
        TotpSecret: totpSecret,
        OtpRecoverCode: otpRecoverCode,
        OtpRecoderCodeExpire: otpRecoverCodeExpire,
        OtpDeleteCode: otpDeleteCode,
        OtpDeleteCodeExpire: otpDeleteCodeExpire,
      }
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
  return obj.(Identity), true, nil
}

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

func SendAnEmailForIdentity(smtpConfig SMTPConfig, identity Identity, anEmail AnEmail) (bool, error) {

  from := mail.Address{smtpConfig.Sender.Name, smtpConfig.Sender.Email}
  to := mail.Address{identity.Name, identity.Email}

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
