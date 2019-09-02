package identities

import (
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "idp/config"
  "idp/environment"
  "idp/gateway/idp"
)

type RecoverRequest struct {
  Id              string            `json:"id" binding:"required"`
}

type RecoverResponse struct {
  Id              string          `json:"id" binding:"required"`
  RedirectTo      string          `json:"redirect_to" binding:"required"`
}

type RecoverTemplateData struct {
  Name string
  VerificationCode string
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

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Id) // FIXME do not return a list of identities!
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    if identities == nil {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return;
    }

    // Found identity prepare to send recover email
    identity := identities[0];

    sender := idp.SMTPSender{
      Name: config.GetString("recover.sender.name"),
      Email: config.GetString("recover.sender.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    recoverChallenge, err := idp.CreateRecoverChallenge(config.GetString("recover.link"), identity, 60 * 5) // Fixme configfy 60*5
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    hashedCode, err := idp.CreatePassword(recoverChallenge.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := idp.Identity{
      Id: identity.Id,
      OtpRecoverCode: hashedCode,
      OtpRecoderCodeExpire: recoverChallenge.Expire,
    }
    updatedIdentity, err := idp.UpdateOtpRecoverCode(env.Driver, n)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{
      "verification_code": recoverChallenge.VerificationCode,
    }).Debug("VERIFICATION CODE - DO NOT DO THIS IN PRODUCTION");

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
      Name: updatedIdentity.Id,
      VerificationCode: recoverChallenge.VerificationCode,
    }

    var tpl bytes.Buffer
    if err := t.Execute(&tpl, data); err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    mail := idp.AnEmail{
      Subject: recoverSubject,
      Body: tpl.String(),
    }

    _, err = idp.SendAnEmailForIdentity(smtpConfig, updatedIdentity, mail)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": updatedIdentity.Id,
        "file": recoverTemplateFile,
      }).Debug("Failed to send recover mail")
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    recoverResponse := RecoverResponse{
      Id: updatedIdentity.Id,
      RedirectTo: recoverChallenge.RedirectTo,
    }
    log.WithFields(logrus.Fields{
      "id": recoverResponse.Id,
      "redirect_to": recoverResponse.RedirectTo,
    }).Debug("Recover mail send")
    c.JSON(http.StatusOK, recoverResponse)
  }
  return gin.HandlerFunc(fn)
}
