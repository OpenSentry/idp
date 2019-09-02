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

type IdentitiesRequest struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type IdentitiesResponse struct {
  Id            string          `json:"id" binding:"required"`
  Password      string          `json:"password" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type DeleteRequest struct {
  Id              string            `json:"id" binding:"required"`
}

type DeleteResponse struct {
  Id              string          `json:"id" binding:"required"`
  RedirectTo      string          `json:"redirect_to" binding:"required"`
}

type DeleteTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func GetCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetCollection",
    })

    id, _ := c.GetQuery("id") // From input

    s, _ := c.Get("sub") // From access token
    subject := s.(string)

    if id == "" && subject == "" {
      c.JSON(http.StatusNotFound, gin.H{
        "error": "Not found",
      })
      c.Abort()
      return;
    }

    if subject != "" && id != "" && subject != id {
      c.JSON(http.StatusForbidden, gin.H{
        "error": "Not allowed. Hint: access token does not match id parameter.",
      })
      c.Abort()
      return;
    }

    if subject == "" && id != "" {
      subject = id
    }

    identityList, err := idp.FetchIdentitiesForSub(env.Driver, subject)
    if err != nil {
      log.WithFields(logrus.Fields{"sub": subject}).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Identity"})
      c.Abort()
      return
    }

    if len(identityList) > 0 {
      n := identityList[0]
      if subject == n.Id {
        c.JSON(http.StatusOK, IdentitiesResponse{
          Id: n.Id,
          Name: n.Name,
          Email: n.Email,
          Password: n.Password,
        })
        return
      }
    }

    // Deny by default
    c.JSON(http.StatusNotFound, gin.H{
      "error": "Not found",
    })
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func PostCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostCollection",
    })

    var input IdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{"id":input.Id}).Debug("Creating identity")
    log.Debug(env.BannedUsernames[input.Id])

    if env.BannedUsernames[input.Id] == true {
      c.JSON(http.StatusNotFound, gin.H{"error": "Id is bannned"})
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
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
      Password: hashedPassword,
    }
    identityList, err := idp.CreateIdentities(env.Driver, newIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := identityList[0]

    c.JSON(http.StatusOK, IdentitiesResponse{
      Id: n.Id,
      Name: n.Name,
      Email: n.Email,
      Password: n.Password,
    })
  }
  return gin.HandlerFunc(fn)
}

func PutCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Warning: Do not log user passwords!
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutCollection",
    })

    var input IdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    updateIdentity := idp.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
    }
    identityList, err := idp.UpdateIdentities(env.Driver, updateIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := identityList[0]

    c.JSON(http.StatusOK, IdentitiesResponse{
      Id: n.Id,
      Name: n.Name,
      Email: n.Email,
      Password: n.Password,
    })
  }
  return gin.HandlerFunc(fn)
}

func DeleteCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteCollection",
    })

    var input DeleteRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{"fixme":1}).Debug("Match that access_token.sub matches requested id to delete");

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
      }).Debug("Failed to send delete mail")
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    deleteResponse := DeleteResponse{
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
