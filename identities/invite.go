package identities

import (
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "time"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/dgrijalva/jwt-go"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

type InviteClaims struct {
	OnBehalfOf string `json:"on_behalf_of"`
	jwt.StandardClaims
}

type InviteTemplateData struct {
  OnBehalfOf string
  Email string
  Link string
  InvitationToken string
}

func PostInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvite",
    })

    var input IdentitiesInviteRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == true {

      anInvitation := idp.Invitation{
        Id: identity.Id,
        Email: input.Email,
      }
      invitation, err := idp.CreateInvitation(identity, anInvitation)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      // Create JWT invite token. Beware not to put sensitive data into these as it is publicly visible.
      // Iff data is secret, then persist an annonymous token in db instead and use that as token.
	    expirationTime := time.Now().Add(1 * time.Hour) // FIXME: config invite expire time
      claims := &InviteClaims{
        OnBehalfOf: invitation.Id,
		    StandardClaims: jwt.StandardClaims{
          Issuer: config.GetString("idp.public.issuer"),
          Audience: config.GetString("idp.public.issuer"),
          Subject: invitation.Email,
          ExpiresAt: expirationTime.Unix(),
        },
	    }

      token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
      jwtInvitation, err := token.SignedString(env.IssuerSignKey)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      sender := idp.SMTPSender{
        Name: config.GetString("invite.sender.name"),
        Email: config.GetString("invite.sender.email"),
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
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      t := template.Must(template.New(emailTemplateFile).Parse(string(tplEmail)))

      data := InviteTemplateData{
        OnBehalfOf: identity.Name,
        Email: input.Email,
        Link: config.GetString("invite.link") + "?invitation=" + jwtInvitation,
        InvitationToken: jwtInvitation,
      }

      var tpl bytes.Buffer
      if err := t.Execute(&tpl, data); err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      mail := idp.AnEmail{
        Subject: emailSubject,
        Body: tpl.String(),
      }

      _, err = idp.SendAnEmailForIdentity(smtpConfig, identity, mail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "id": identity.Id,
          "file": emailTemplateFile,
        }).Debug("Failed to send invite mail")
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      response := IdentitiesInviteResponse{
        Invitation: jwtInvitation,
      }
      log.WithFields(logrus.Fields{
        "invitation": response.Invitation,
      }).Debug("Invite send")
      c.JSON(http.StatusOK, response)
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
