package identities

import (
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

type RecoverTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func PostRecover(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecover",
    })

    var input IdentitiesRecoverRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    humans, err := idp.FetchHumansById(env.Driver, []string{input.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if humans == nil {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }
    human := humans[0]

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

    recoverChallenge, err := idp.CreateRecoverChallenge(config.GetString("recover.link"), human, 60 * 5) // Fixme configfy 60*5
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    hashedCode, err := idp.CreatePassword(recoverChallenge.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    n := idp.Human{
      Identity: idp.Identity{
        Id: human.Id,
      },
      OtpRecoverCode: hashedCode,
      OtpRecoverCodeExpire: recoverChallenge.Expire,
    }
    updatedIdentity, err := idp.UpdateOtpRecoverCode(env.Driver, n)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
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
      c.AbortWithStatus(http.StatusInternalServerError)
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
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    mail := idp.AnEmail{
      Subject: recoverSubject,
      Body: tpl.String(),
    }

    _, err = idp.SendAnEmailToHuman(smtpConfig, updatedIdentity, mail)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": updatedIdentity.Id,
        "file": recoverTemplateFile,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    recoverResponse := IdentitiesRecoverResponse{
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
