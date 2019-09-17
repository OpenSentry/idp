package invites

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

func PostInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostIdentities",
    })

    var input InviteCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // Sanity check. Inviter
    inviterIdentity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if exists == false {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }

    // Look for invited identity using email
    var isAnonymousInvite bool = false
    invitedIdentity, exists, err := idp.FetchIdentityByEmail(env.Driver, input.Email)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
    }
    isAnonymousInvite = !exists

    // Sanity check. Granted scopes
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
        c.AbortWithStatus(http.StatusInternalServerError)
        return
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
    invite, err := idp.CreateInvite(env.Driver, inviterIdentity, inviteRequest)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": inviterIdentity.Id,
        "email": inviteRequest.Email,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
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
      c.AbortWithStatus(http.StatusInternalServerError)
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
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    mail := idp.AnEmail{
      Subject: emailSubject,
      Body: tpl.String(),
    }

    if isAnonymousInvite == true {
      _, err = idp.SendAnEmailToAnonymous(smtpConfig, invite.Email, invite.Email, mail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "email": invite.Email,
          "file": emailTemplateFile,
        }).Debug("Failed to send invite mail")
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
    } else {
      _, err = idp.SendAnEmailToIdentity(smtpConfig, invitedIdentity, mail)
      if err != nil {
        log.WithFields(logrus.Fields{
          "id": invitedIdentity.Id,
          "file": emailTemplateFile,
        }).Debug("Failed to send invite mail")
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
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
  return gin.HandlerFunc(fn)
}

func GetInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetInvites",
    })

    var err error

    var request InviteReadRequest
    err = c.BindJSON(&request)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    if request.Id == "" {

      // FIXME: Do bulk request

    } else {

      invite, exists, err := idp.FetchInviteById(env.Driver, request.Id)
      if err != nil {
        log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      if exists == true {
        c.JSON(http.StatusOK, InviteReadResponse{ marshalInviteToInviteResponse(invite) })
        return
      }

    }

    // Deny by default
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
    c.Abort()
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
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    invite, exists, err := idp.FetchInviteById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == true {

      // Created granted relations as specified in the invite
      // Create follow relations as specified in the invite
      accept, err := idp.AcceptInvite(env.Driver, invite)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
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
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
  }
  return gin.HandlerFunc(fn)
}

func marshalInviteToInviteResponse(invite idp.Invite) *InviteResponse {
  return &InviteResponse{
    Id: invite.Id,
    Email: invite.Email,
    Username: invite.Username,
    GrantedScopes: invite.GrantedScopes,
    FollowIdentities: invite.FollowIdentities,
    TTL: invite.ExpiresInSeconds,
    IssuedAt: invite.IssuedAt,
    ExpiresAt: invite.ExpiresAt,
    InviterId: invite.InviterIdentityId,
    InvitedId: invite.InvitedIdentityId,
  }
}