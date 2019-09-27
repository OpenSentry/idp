package humans

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
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
)

type DeleteTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func GetHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetHumans",
    })

    var requests []client.ReadHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request) {
      var humans []idp.Human

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadHumansRequest
          r = request.Request.(client.ReadHumansRequest)
          humans = append(humans, idp.Identity{ Id:r.Id, Email:r.Email, Username:r.Username })
        }
      }

      dbHumans, err := idp.FetchHumans(env.Driver, humans)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatusJSON(http.StatusInternalServerError)
        return
      }

      // TODO: To decrease memory usage fix to use a pointer instead.
      var mapHumans map[string]idp.Human
      if ( iRequests[0] != nil ) {
        for _, human := range dbHumans {
          mapHumans[human.Id] = human
        }
      }

      for _, request := range iRequests {
        var r client.ReadHumansRequest
        if request.Request == nil {

          // The empty fetch
          for _, i := range dbHumans {
            ok = append(ok, client.Human{
              Id: i.Id,
              Username: i.Usermame,
              Password: i.Password,
              Name: i.Name,
              Email: i.Email,
              AllowLogin: i.AllowLogin,
              TotpRequired: i.TotpRequired,
              TotpSecret: i.TotpSecret,
              OtpRecoverCode: i.OtpRecoverCode,
              OtpRecoverCodeExpire: i.OtpRecoverCodeExpire,
              OtpDeleteCode: i.OtpDeleteCode,
              OtpDeleteCodeExpire: i.OtpDeleteCodeExpire,
            })
          }
          var response client.ReadIdentitiesResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r = request.Request.(client.ReadHumansRequest)

          var i = mapHumans[r.Id]
          if i != nil {
            ok := client.Human{
              Id: i.Id,
              Username: i.Usermame,
              Password: i.Password,
              Name: i.Name,
              Email: i.Email,
              AllowLogin: i.AllowLogin,
              TotpRequired: i.TotpRequired,
              TotpSecret: i.TotpSecret,
              OtpRecoverCode: i.OtpRecoverCode,
              OtpRecoverCodeExpire: i.OtpRecoverCodeExpire,
              OtpDeleteCode: i.OtpDeleteCode,
              OtpDeleteCodeExpire: i.OtpDeleteCodeExpire,
            }
            var response client.ReadIdentitiesResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = ok
            request.Response = response
            continue
          }

        }

        // Deny by default
        var response client.ReadIdentitiesResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = 0 // FIXME: Hent utils fra aap og bruge NewClientError( med coden som i aap)
        request.Response = response
        continue

      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PostHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostHumans",
    })

    var requests []client.HumansCreateRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.HumansCreateRequest)

        if env.BannedUsernames[r.Username] == true {
          c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Username is banned"})
          return
        }

        hashedPassword, err := idp.CreatePassword(r.Password)
        if err != nil {
          c.AbortWithStatus(http.StatusInternalServerError)
          return
        }

        newHuman := idp.Human{
          Username: r.Username,
          Name: r.Name,
          Email: r.Email,
          Password: hashedPassword,
          AllowLogin: true,
        }
        human, err := idp.CreateHuman(env.Driver, newHuman)
        if err != nil {
          log.WithFields(logrus.Fields{ "username": newHuman.Username, "name": newHuman.Name, "email": newHuman.Email, "password": newHuman.Password, newHuman.AllowLogin }).Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if human != nil {
         response := client.HumansCreateResponse{Ok: ok}
         response.Index = request.Index
         response.Status = http.StatusOK
         request.Response = human
         continue;
        }

        // Deny by default
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})
    c.JSON(http.StatusOK, responses)
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

func marshalHumanToHumanResponse(identity idp.Human) *HumansResponse {
  return &HumansResponse{
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



