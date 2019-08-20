package idpapi

import (
  "crypto/aes"
  "crypto/cipher"
  "crypto/rand"
  "crypto/hmac"
  "crypto/sha256"
  "encoding/base64"
  "encoding/hex"
  "errors"
  "io"
  "net"
  "net/mail"
  "net/smtp"
  "strings"
  "crypto/tls"
  "fmt"
  "golang.org/x/crypto/bcrypt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "github.com/pquerna/otp/totp"
)

type Identity struct {
  Id         string `json:"id" binding:"required"`
  Name       string `json:"name"`
  Email      string `json:"email"`
  Password   string `json:"password"`
  Require2Fa bool   `json:"require_2fa"`
  Secret2Fa  string `json:"secret"`
}

type PasscodeChallenge struct {
  Challenge  string `json:"challenge" binding:"required"`
  Id         string `json:"id" binding:"required"`
  Signature  string `json:"id" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
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

func ValidatePasscode(passcode string, secret string) (bool, error) {
  valid := totp.Validate(passcode, secret)
  return valid, nil
}

func CreatePasscodeChallenge(url string, challenge string, id string, secret string) PasscodeChallenge {

  redirectTo := url + "?login_challenge=" + challenge + "&id=" + id
  h := hmac.New(sha256.New, []byte(secret))
  h.Write([]byte(redirectTo))
  sha := hex.EncodeToString(h.Sum(nil))
  return PasscodeChallenge{
    Challenge: challenge,
    Id: id,
    Signature: sha,
    RedirectTo: redirectTo + "&sig=" + sha,
  }
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

func UpdateTwoFactor(driver neo4j.Driver, identity Identity) (Identity, error) {
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
    cypher := "MATCH (i:Identity {sub:$sub}) SET i.require_2fa=$required, i.secret_2fa=$secret RETURN i.sub, i.password, i.name, i.email, i.require_2fa, i.secret_2fa"
    params := map[string]interface{}{"sub": identity.Id, "required": identity.Require2Fa, "secret": identity.Secret2Fa}
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
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)
      require2Fa := record.GetByIndex(4).(bool)
      secret2Fa := record.GetByIndex(5).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
        Require2Fa: require2Fa,
        Secret2Fa: secret2Fa,
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
    cypher := "MATCH (i:Identity {sub:$sub}) SET i.password=$password RETURN i.sub, i.password, i.name, i.email, i.require_2fa, i.secret_2fa"
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
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)
      require2Fa := record.GetByIndex(4).(bool)
      secret2Fa := record.GetByIndex(5).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
        Require2Fa: require2Fa,
        Secret2Fa: secret2Fa,
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
      CREATE (i:Identity {sub:$sub, password:$password, name:$name, email:$email, require_2fa:false, secret_2fa:""}) RETURN i.sub, i.password, i.name, i.email, i.require_2fa, i.secret_2fa
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
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)
      require2Fa := record.GetByIndex(4).(bool)
      secret2Fa := record.GetByIndex(5).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
        Require2Fa: require2Fa,
        Secret2Fa: secret2Fa,
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
    cypher := "MATCH (i:Identity {sub:$sub}) WITH i SET i.name=$name, i.email=$email RETURN i.sub, i.password, i.name, i.email, i.require_2fa, i.secret_2fa"
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
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)
      require2Fa := record.GetByIndex(4).(bool)
      secret2Fa := record.GetByIndex(5).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
        Require2Fa: require2Fa,
        Secret2Fa: secret2Fa,
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

// https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
func FetchIdentitiesForSub(driver neo4j.Driver, sub string) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := "MATCH (i:Identity {sub: $sub}) RETURN i.sub, i.password, i.name, i.email, i.require_2fa, i.secret_2fa ORDER BY i.sub"
    params := map[string]interface{}{"sub": sub}
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
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)
      require2Fa := record.GetByIndex(4).(bool)
      secret2Fa := record.GetByIndex(5).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
        Require2Fa: require2Fa,
        Secret2Fa: secret2Fa,
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

type RecoverMail struct {
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

func SendRecoverMailForIdentity(smtpConfig SMTPConfig, identity Identity, recoverMail RecoverMail) (bool, error) {

  from := mail.Address{smtpConfig.Sender.Name, smtpConfig.Sender.Email}
  to := mail.Address{identity.Name, identity.Email}

  subject := recoverMail.Subject
  body := recoverMail.Body

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
