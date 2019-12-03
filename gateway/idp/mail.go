package idp

import (
  "encoding/base64"
  "net"
  "net/mail"
  "net/smtp"
  "strings"
  "crypto/tls"
  "fmt"
  "text/template"
  "io/ioutil"
  "bytes"
)

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

func SendEmailUsingTemplate(smtpConfig SMTPConfig, name string, email string, subject string, templateFile string, data interface{}) (bool, error) {
  tplRecover, err := ioutil.ReadFile(templateFile)
  if err != nil {
    return false, err
  }

  t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

  var tpl bytes.Buffer
  if err := t.Execute(&tpl, data); err != nil {
    return false, err
  }

  return SendEmail(smtpConfig, name, email, subject, tpl.String())
}

func SendEmail(smtpConfig SMTPConfig, name string, email string, subject string, body string) (bool, error) {

  from := mail.Address{smtpConfig.Sender.Name, smtpConfig.Sender.Email}
  to := mail.Address{name, email}

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
  // auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, host)

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
  // if err := c.Auth(auth); err != nil {
  //   return false, err
  // }

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