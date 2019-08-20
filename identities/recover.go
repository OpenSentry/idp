package identities

import (
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpapi"
)

type RecoverRequest struct {
  Id              string            `json:"id" binding:"required"`
}

type RecoverResponse struct {
  Id              string          `json:"id" binding:"required"`
  RecoverMethod   string          `json:"recover_method" binding:"required"`
  Email           string          `json:"email" binding:"required"`
}

type RecoverTemplateData struct {
  Name string
  RecoverLink string
  Sender string
}

func PostRecover(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecover",
    })

    var input RecoverRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    identities, err := idpapi.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }
    if identities != nil {
      identity := identities[0]; // FIXME do not return a list of identities!

      sender := idpapi.SMTPSender{
        Name: config.GetString("recover.sender.name"),
        Email: config.GetString("recover.sender.email"),
      }

      smtpConfig := idpapi.SMTPConfig{
        Host: config.GetString("mail.smtp.host"),
        Username: config.GetString("mail.smtp.user"),
        Password: config.GetString("mail.smtp.password"),
        Sender: sender,
        SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
      }

      recoverLink := config.GetString("recover.link")
      recoverChallenge := "1234"

      recoverTemplateFile := config.GetString("recover.template.email.file")
      recoverSubject := config.GetString("recover.template.email.subject")

      tplRecover, err := ioutil.ReadFile(recoverTemplateFile)
      if err != nil {
        log.WithFields(logrus.Fields{
          "file": recoverTemplateFile,
        }).Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      t := template.Must(template.New(recoverTemplateFile).Parse(string(tplRecover)))

      data := RecoverTemplateData{
        Sender: sender.Name,
        Name: input.Id,
        RecoverLink: recoverLink + "?recover_challenge=" + recoverChallenge,
      }

      var tpl bytes.Buffer
      if err := t.Execute(&tpl, data); err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      recoverMail := idpapi.RecoverMail{
        Subject: recoverSubject,
        Body: tpl.String(),
      }

      _, err = idpapi.SendRecoverMailForIdentity(smtpConfig, identity, recoverMail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "id": identity.Id,
          "file": recoverTemplateFile,
        }).Debug("Failed to send recover mail")
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      recoverResponse := RecoverResponse{
        Id: identity.Id,
        Email: identity.Email,
        RecoverMethod: "email",
      }
      log.WithFields(logrus.Fields{
        "id": recoverResponse.Id,
        "recover_method": recoverResponse.RecoverMethod,
        "email": recoverResponse.Email,
      }).Debug("Recover mail send")
      c.JSON(http.StatusOK, recoverResponse)
      c.Abort()
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
    return;
  }
  return gin.HandlerFunc(fn)
}
