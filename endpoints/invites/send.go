package invites

import (
  "text/template"
  "io/ioutil"
  "bytes"
  "net/url"

  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  _ "github.com/charmixer/idp/client"
)

type InviteSendRequest struct {
  Id string `json:"id" binding:"required"`
}

type InviteSendResponse struct {
  Id string `json:"id" binding:"required"`
}

type InviteTemplateData struct {
  Id string
  InvitedBy string
  Email string
  InvitationUrl string
  IdentityProvider string
}

func PostSend(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostSend",
    })

    var input InviteSendRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    invites, err := idp.FetchInvitesById(env.Driver, []string{input.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if invites == nil {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invite not found."})
      return
    }
    invite := invites[0]

    invitedByIdentities, err := idp.FetchHumansById(env.Driver, []string{invite.InvitedBy.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if invitedByIdentities == nil {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: invited_by"})
      return
    }
    invitedByIdentity := invitedByIdentities[0]

    invitedIdentities, err := idp.FetchHumansById(env.Driver, []string{invite.Invited.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    isAnonymousInvite := invitedIdentities == nil

    var invitedIdentity idp.Human
    if invitedIdentities != nil {
      invitedIdentity = invitedIdentities[0]
    }

    sender := idp.SMTPSender{
      Name: config.GetString("provider.name"),
      Email: config.GetString("provider.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    emailTemplateFile := config.GetString("invite.template.email.file")
    emailSubject := config.GetString("invite.template.email.subject")

    tplEmail, err := ioutil.ReadFile(emailTemplateFile)
    if err != nil {
      log.WithFields(logrus.Fields{
        "file": emailTemplateFile,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    t := template.Must(template.New(emailTemplateFile).Parse(string(tplEmail)))

    u, err := url.Parse( config.GetString("invite.url") )
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    q := u.Query()
    q.Add("id", invite.Id)
    u.RawQuery = q.Encode()

    data := InviteTemplateData{
      Id: invite.Id,
      InvitedBy: invitedByIdentity.Name,
      Email: invite.SentTo.Email,
      InvitationUrl: u.String(),
      IdentityProvider: config.GetString("provider.name"),
    }

    var tpl bytes.Buffer
    if err := t.Execute(&tpl, data); err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    mail := idp.AnEmail{
      Subject: emailSubject,
      Body: tpl.String(),
    }

    if isAnonymousInvite == true {

      _, err = idp.SendAnEmailToAnonymous(smtpConfig, invite.SentTo.Email, invite.SentTo.Email, mail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "email": invite.SentTo.Email,
          "file": emailTemplateFile,
        }).Debug("Failed to send invite mail")
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

    } else {

      _, err = idp.SendAnEmailToHuman(smtpConfig, invitedIdentity, mail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "id": invitedIdentity.Id,
          "file": emailTemplateFile,
        }).Debug("Failed to send invite mail")
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

    }

    response := InviteSendResponse{
      Id: invite.Id,
    }
    log.WithFields(logrus.Fields{
      "id": response.Id,
    }).Debug("Recover mail send")
    c.JSON(http.StatusOK, response)
  }
  return gin.HandlerFunc(fn)
}