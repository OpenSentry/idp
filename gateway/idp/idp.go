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
  "github.com/pquerna/otp/totp"
)

type RecoverChallenge struct {
  Id         string
  Code       string
  Expire     int64
  RedirectTo string
}

type DeleteChallenge struct {
  Id         string
  Code       string
  Expire     int64
  RedirectTo string
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

func CreateDeleteChallenge(url string, identity Human, challengeTimeoutInSeconds int64) (DeleteChallenge, error) {
  code, err := GenerateRandomDigits(6);
  if err != nil {
    return DeleteChallenge{}, err
  }

  timeout := time.Duration(challengeTimeoutInSeconds)
  expirationTime := time.Now().Add(timeout * time.Second)
  expiresAt := expirationTime.Unix()
  redirectTo := url

  return DeleteChallenge{
    Id: identity.Id,
    Code: code,
    Expire: expiresAt,
    RedirectTo: redirectTo,
  }, nil
}

func CreateRecoverChallenge(url string, identity Human, challengeTimeoutInSeconds int64) (RecoverChallenge, error) {
  code, err := GenerateRandomDigits(6);
  if err != nil {
    return RecoverChallenge{}, err
  }

  timeout := time.Duration(challengeTimeoutInSeconds)
  expirationTime := time.Now().Add(timeout * time.Second)
  expiresAt := expirationTime.Unix()
  redirectTo := url

  return RecoverChallenge{
    Id: identity.Id,
    Code: code,
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

func SendAnEmailToHuman(smtpConfig SMTPConfig, human Human, anEmail AnEmail) (bool, error) {
  return sendAnEmail(smtpConfig, human.Name, human.Email, anEmail)
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
