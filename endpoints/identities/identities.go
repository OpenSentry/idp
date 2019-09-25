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

func GetIdentities(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetIdentities",
    })

    var err error

    var request IdentitiesReadRequest
    err = c.BindJSON(&request)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var humans []idp.Human

    if request.Id == "" {

      if request.Username != "" {
        humans, err = idp.FetchHumansByUsername(env.Driver, []string{request.Username})
        if err != nil {
          log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
          c.AbortWithStatus(http.StatusInternalServerError)
          return
        }
      }

      if len(humans) <= 0 && request.Email != "" {
        humans, err = idp.FetchHumansByEmail(env.Driver, []string{request.Email})
        if err != nil {
          log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
          c.AbortWithStatus(http.StatusInternalServerError)
          return
        }
      }

    } else {
      humans, err = idp.FetchHumansById(env.Driver, []string{request.Id})
      if err != nil {
        log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
    }

    if len(humans) > 0 {
      c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(humans[0]) })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": request.Id, "username": request.Username, "email": request.Email}).Debug("Identity not found")
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func PostIdentities(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostIdentities",
    })

    var input IdentitiesCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    if input.Username == "" {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing username"})
      return
    }

    if env.BannedUsernames[input.Username] == true {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Username is banned"})
      return
    }

    hashedPassword, err := idp.CreatePassword(input.Password)
    if err != nil {
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    newHuman := idp.Human{
      Username: input.Username,
      Name: input.Name,
      Email: input.Email,
      Password: hashedPassword,
    }
    human, err := idp.CreateHuman(env.Driver, newHuman)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
      return
    }

    c.JSON(http.StatusOK, IdentitiesCreateResponse{ marshalIdentityToIdentityResponse(human) })
  }
  return gin.HandlerFunc(fn)
}

func PutIdentities(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Warning: Do not log user passwords!
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutIdentities",
    })

    var input IdentitiesUpdateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // Inspect access token for subject
    s, _ := c.Get("sub") // Middleware delivers access_token.id_token.sub
    subject := s.(string)

    // Sanity check. Require subject from access token
    if subject == "" {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Missing subject in access_token"})
      return
    }

    // Sanity check. Identity exists
    if input.Id == "" {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }

    // Sanity check. Access token subject and Identity.Subject must match.
    if subject != input.Id {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Not allowed. Hint: Access token subject and Identity.Id does not match"})
      return
    }

    updateHuman := idp.Human{
      Identity: idp.Identity{
        Id: input.Id,
      },
      Name: input.Name,
      Email: input.Email,
    }
    human, err := idp.UpdateHuman(env.Driver, updateHuman)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    c.JSON(http.StatusOK, IdentitiesUpdateResponse{ marshalIdentityToIdentityResponse(human) })
  }
  return gin.HandlerFunc(fn)
}

func DeleteIdentities(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteIdentities",
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

    identities, err := idp.FetchIdentitiesById(env.Driver, []string{input.Id})
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
    identity := identities[0]

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

    n := idp.Human{
      Identity: idp.Identity{
        Id: identity.Id,
        OtpDeleteCode: hashedCode,
        OtpDeleteCodeExpire: deleteChallenge.Expire,
      },
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

    _, err = idp.SendAnEmailToHuman(smtpConfig, updatedIdentity, anEmail)
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

func marshalIdentityToIdentityResponse(identity idp.Human) *IdentitiesResponse {
  return &IdentitiesResponse{
    Id:                   identity.Id,
    Username:             identity.Username,
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
