package identities

import (
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "strings"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

type Scope struct {
  Name string
  Title string
  Description string
}

type Follow struct {
  Id string
  Name string
  Introduction string
  PublicProfileUrl string
}

type InviteTemplateData struct {
  IdentityProvider string
  OnBehalfOf string
  Email string
  InvitationUrl string
  InvitationToken string
  Scopes []Scope
  Follows []Follow
}

func GetInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetInvite",
    })

    var err error

    var request IdentitiesInviteReadRequest
    err = c.BindJSON(&request)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    invite, exists, err := idp.FetchInviteById(env.Driver, request.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == true {

      response := IdentitiesInviteReadResponse{
        IdentitiesInviteResponse: &IdentitiesInviteResponse{
          Id: invite.Id,
          Email: invite.Email,
        },
      }
      c.JSON(http.StatusOK, response)
      return

    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": request.Id}).Info("Invite not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Invite not found"})

  }
  return gin.HandlerFunc(fn)
}

func PutInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutInvite",
    })

    var input IdentitiesInviteUpdateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    invite, exists, err := idp.FetchInviteById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == true {

      // Created granted relations as specified in the invite
      // Create follow relations as specified in the invite
      accept, err := idp.AcceptInvite(env.Driver, invite)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return;
      }

      response := IdentitiesInviteUpdateResponse{
        IdentitiesInviteResponse: &IdentitiesInviteResponse{
          Id: accept.Id,
        },
      }
      log.WithFields(logrus.Fields{
        "id": accept.Id,
      }).Debug("Invite accepted")
      c.JSON(http.StatusOK, response)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Invite not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
  }
  return gin.HandlerFunc(fn)
}

func PostInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvite",
    })

    var input IdentitiesInviteCreateRequest
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

      log.WithFields(logrus.Fields{"fixme": 1}).Debug("Call aap to check if scopes exists")
      var scopes []Scope
      for _, scope := range input.GrantedScopes {
        scopes = append(scopes, Scope{
          Name: scope,
          Title: "A scope",
          Description: "Se alt det du kan",
        })
      }

      // Find all identities requested to be followed, ignore non existing ones.
      var followIdentities []string
      var follows []Follow
      for _, identityId := range input.PleaseFollow {

        followIdentity, exists, err := idp.FetchIdentityById(env.Driver, identityId)
        if err != nil {
          log.Debug(err.Error())
          c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
          c.Abort()
          return;
        }

        if exists == true {
          followIdentities = append(followIdentities, followIdentity.Id)
          follows = append(follows, Follow{
            Id: followIdentity.Id,
            Name: followIdentity.Name,
            Introduction: "Jeg er en spændende person eller en client eller en resource server",
            PublicProfileUrl: "https://id.localhost/profile?id" + followIdentity.Id,
          })
        }

      }

      log.WithFields(logrus.Fields{"fixme": 1}).Debug("Put invite expiration into config")
      var expiresInSeconds int64 = 60*60*24 // 24 hours
      inviteRequest := idp.Invite{
        Email: input.Email,
        GrantedScopes: strings.Join(input.GrantedScopes, " "),
        FollowIdentities: strings.Join(followIdentities, " "),
        ExpiresInSeconds: expiresInSeconds,
      }
      invite, err := idp.CreateInvite(env.Driver, identity, inviteRequest)
      if err != nil {
        log.WithFields(logrus.Fields{
          "id": identity.Id,
          "email": inviteRequest.Email,
        }).Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
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
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      t := template.Must(template.New(emailTemplateFile).Parse(string(tplEmail)))

      data := InviteTemplateData{
        OnBehalfOf: invite.InviterIdentityId,
        Email: invite.Email,
        IdentityProvider: config.GetString("provider.name"),
        InvitationUrl: config.GetString("invite.url"),
        Scopes: scopes,
        Follows: follows,
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

      response := IdentitiesInviteCreateResponse{
        IdentitiesInviteResponse: &IdentitiesInviteResponse{
          Id: invite.Id,
        },
      }
      log.WithFields(logrus.Fields{
        "id": response.Id,
      }).Debug("Invite send")
      c.JSON(http.StatusOK, response)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
