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

type DeleteTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func GetCollection(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetCollection",
    })

    var err error

    var request IdentitiesReadRequest
    err = c.BindJSON(&request)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Inspect access token for subject
    /*s, _ := c.Get("sub") // Middleware delivers access_token.id_token.sub
    subject := s.(string)
    log.WithFields(logrus.Fields{"sub": subject}).Debug("Found subject in access token")

    // Sanity check. Require subject from access token
    if subject == "" {
      c.JSON(http.StatusForbidden, gin.H{"error": "Missing subject in access_token"})
      c.Abort()
      return
    }*/

    var identity idp.Identity
    var exists bool

    log.WithFields(logrus.Fields{"id":request.Id, "subject":request.Subject, "email":request.Email}).Debug("Received read:identity request")

    if request.Id == "" {

      // Look for identity id using either subject or email
      if request.Subject != "" {
        identity, exists, err = idp.FetchIdentityBySubject(env.Driver, request.Subject)
        if err != nil {
          log.WithFields(logrus.Fields{"sub": request.Subject}).Debug(err.Error())
          c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
          c.Abort()
          return
        }
      }

      if identity.Id == "" && request.Email != "" {
        identity, exists, err = idp.FetchIdentityByEmail(env.Driver, request.Email)
        if err != nil {
          log.WithFields(logrus.Fields{"email": request.Email}).Debug(err.Error())
          c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
          c.Abort()
          return
        }
      }

    } else {

      identity, exists, err = idp.FetchIdentityById(env.Driver, request.Id)
      if err != nil {
        log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

    }

    // Sanity check. Access token subject and Identity.Subject must match.
    //if subject != identity.Id {
    //  c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed. Hint: Access token subject and Identity.Id does not match"})
    //  c.Abort()
    //  return
    //}

    if exists == true {
      c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(identity) })
      return
    }

    // Deny by default
    c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func PostCollection(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostCollection",
    })

    var input IdentitiesCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    if input.Subject == "" {
      c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sub"})
      c.Abort()
      return
    }

    if env.BannedUsernames[input.Subject] == true {
      c.JSON(http.StatusForbidden, gin.H{"error": "Subject is banned"})
      c.Abort()
      return
    }

    hashedPassword, err := idp.CreatePassword(input.Password)
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    newIdentity := idp.Identity{
      Subject: input.Subject,
      Name: input.Name,
      Email: input.Email,
      Password: hashedPassword,
    }
    identity, err := idp.CreateIdentity(env.Driver, newIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    c.JSON(http.StatusOK, IdentitiesCreateResponse{ marshalIdentityToIdentityResponse(identity) })
  }
  return gin.HandlerFunc(fn)
}

func PutCollection(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Warning: Do not log user passwords!
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutCollection",
    })

    var input IdentitiesUpdateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Inspect access token for subject
    s, _ := c.Get("sub") // Middleware delivers access_token.id_token.sub
    subject := s.(string)

    // Sanity check. Require subject from access token
    if subject == "" {
      c.JSON(http.StatusForbidden, gin.H{"error": "Missing subject in access_token"})
      c.Abort()
      return
    }

    // Sanity check. Identity exists
    if input.Id == "" {
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort()
      return
    }

    // Sanity check. Access token subject and Identity.Subject must match.
    if subject != input.Id {
      c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed. Hint: Access token subject and Identity.Id does not match"})
      c.Abort()
      return
    }

    updateIdentity := idp.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
    }
    identity, err := idp.UpdateIdentity(env.Driver, updateIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    c.JSON(http.StatusOK, IdentitiesUpdateResponse{ marshalIdentityToIdentityResponse(identity) })
  }
  return gin.HandlerFunc(fn)
}

func DeleteCollection(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteCollection",
    })

    var input IdentitiesDeleteRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Inspect access token for subject
    s, _ := c.Get("sub") // Middleware delivers access_token.id_token.sub
    subject := s.(string)

    // Sanity check. Require subject from access token
    if subject == "" {
      c.JSON(http.StatusForbidden, gin.H{"error": "Missing subject in access_token"})
      c.Abort()
      return
    }

    // Sanity check. Identity exists
    if input.Id == "" {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort()
      return
    }

    // Sanity check. Access token subject and Identity.Subject must match.
    if subject != input.Id {
      c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed. Hint: Access token subject and Identity.Id does not match"})
      c.Abort()
      return
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    if exists == false {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return;
    }

    sender := idp.SMTPSender{
      Name: config.GetString("delete.sender.name"),
      Email: config.GetString("delete.sender.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    deleteChallenge, err := idp.CreateDeleteChallenge(config.GetString("delete.link"), identity, 60 * 5) // Fixme configfy 60*5
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    hashedCode, err := idp.CreatePassword(deleteChallenge.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := idp.Identity{
      Id: identity.Id,
      OtpDeleteCode: hashedCode,
      OtpDeleteCodeExpire: deleteChallenge.Expire,
    }
    updatedIdentity, err := idp.UpdateOtpDeleteCode(env.Driver, n)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{
      "verification_code": deleteChallenge.VerificationCode,
    }).Debug("VERIFICATION CODE - DO NOT DO THIS IN PRODUCTION");

    templateFile := config.GetString("delete.template.email.file")
    emailSubject := config.GetString("delete.template.email.subject")

    tplRecover, err := ioutil.ReadFile(templateFile)
    if err != nil {
      log.WithFields(logrus.Fields{
        "file": templateFile,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

    data := DeleteTemplateData{
      Sender: sender.Name,
      Name: updatedIdentity.Id,
      VerificationCode: deleteChallenge.VerificationCode,
    }

    var tpl bytes.Buffer
    if err := t.Execute(&tpl, data); err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    anEmail := idp.AnEmail{
      Subject: emailSubject,
      Body: tpl.String(),
    }

    _, err = idp.SendAnEmailForIdentity(smtpConfig, updatedIdentity, anEmail)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": updatedIdentity.Id,
        "file": templateFile,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    deleteResponse := IdentitiesDeleteResponse{
      Id: updatedIdentity.Id,
      RedirectTo: deleteChallenge.RedirectTo,
    }
    log.WithFields(logrus.Fields{
      "id": deleteResponse.Id,
      "redirect_to": deleteResponse.RedirectTo,
    }).Debug("Delete mail send")
    c.JSON(http.StatusOK, deleteResponse)
  }
  return gin.HandlerFunc(fn)
}

func marshalIdentityToIdentityResponse(identity idp.Identity) *IdentitiesResponse {
  return &IdentitiesResponse{
    Id:                   identity.Id,
    Subject:              identity.Subject,
    Password:             identity.Password,
    Name:                 identity.Name,
    Email:                identity.Email,
    AllowLogin:           identity.AllowLogin,
    TotpRequired:         identity.TotpRequired,
    TotpSecret:           identity.TotpSecret,
    OtpRecoverCode:       identity.OtpRecoverCode,
    OtpRecoverCodeExpire: identity.OtpRecoverCodeExpire,
    OtpDeleteCode:        identity.OtpDeleteCode,
    OtpDeleteCodeExpire:  identity.OtpDeleteCodeExpire,
  }
}
