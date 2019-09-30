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
  E "github.com/charmixer/idp/client/errors"
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
      var identities []idp.Human

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadHumansRequest
          r = request.Request.(client.ReadHumansRequest)
          identities = append(identities, idp.Human{ Identity: idp.Identity{Id:r.Id}, Email:r.Email, Username:r.Username })
        }
      }

      dbHumans, err := idp.FetchHumans(env.Driver, identities)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      var mapHumans map[string]*idp.Human
      if ( iRequests[0] != nil ) {
        for _, human := range dbHumans {
          mapHumans[human.Id] = &human
        }
      }

      for _, request := range iRequests {

        if request.Request == nil {

          // The empty fetch
          var ok []client.Human
          for _, i := range dbHumans {
            ok = append(ok, client.Human{
              Id:                   i.Id,
              Username:             i.Username,
              Password:             i.Password,
              Name:                 i.Name,
              Email:                i.Email,
              AllowLogin:           i.AllowLogin,
              TotpRequired:         i.TotpRequired,
              TotpSecret:           i.TotpSecret,
              OtpRecoverCode:       i.OtpRecoverCode,
              OtpRecoverCodeExpire: i.OtpRecoverCodeExpire,
              OtpDeleteCode:        i.OtpDeleteCode,
              OtpDeleteCodeExpire:  i.OtpDeleteCodeExpire,
            })
          }
          var response client.ReadHumansResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r := request.Request.(client.ReadHumansRequest)

          var i = mapHumans[r.Id]
          if i != nil {
            ok := []client.Human{ {
                Id:                   i.Id,
                Username:             i.Username,
                Password:             i.Password,
                Name:                 i.Name,
                Email:                i.Email,
                AllowLogin:           i.AllowLogin,
                TotpRequired:         i.TotpRequired,
                TotpSecret:           i.TotpSecret,
                OtpRecoverCode:       i.OtpRecoverCode,
                OtpRecoverCodeExpire: i.OtpRecoverCodeExpire,
                OtpDeleteCode:        i.OtpDeleteCode,
                OtpDeleteCodeExpire:  i.OtpDeleteCodeExpire,
              },
            }
            var response client.ReadHumansResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = ok
            request.Response = response
            continue
          }

        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
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

    var requests []client.CreateHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      // requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.CreateHumansRequest)

        if env.BannedUsernames[r.Username] == true {
          request.Response = utils.NewClientErrorResponse(request.Index, E.USERNAME_BANNED)
          continue
        }

        hashedPassword, err := idp.CreatePassword(r.Password) // @SecurityRisk: Please _NEVER_ log the cleartext password
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
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
          // @SecurityRisk: Please _NEVER_ log the hashed password
          log.WithFields(logrus.Fields{ "username": newHuman.Username, "name": newHuman.Name, "email": newHuman.Email, /* "password": newHuman.Password, */ "allow_login":newHuman.AllowLogin }).Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if human != (idp.Human{}) {
          ok := client.Human{
              Id: human.Id,
              Username: human.Username,
              Password: human.Password,
              Name: human.Name,
              Email: human.Email,
              AllowLogin: human.AllowLogin,
              TotpRequired: human.TotpRequired,
              TotpSecret: human.TotpSecret,
              OtpRecoverCode: human.OtpRecoverCode,
              OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
              OtpDeleteCode: human.OtpDeleteCode,
              OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }
          var response client.CreateHumansResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue
        }

        // Deny by default
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{ "username": newHuman.Username, "name": newHuman.Name, "email": newHuman.Email, /* "password": newHuman.Password, */ "allow_login":newHuman.AllowLogin }).Debug(err.Error())
        request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_CREATED)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PutHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutHumans",
    })

    var requests []client.UpdateHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      //requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.UpdateHumansRequest)

        updateHuman := idp.Human{
          Identity: idp.Identity{
            Id: r.Id,
          },
          Name: r.Name,
          Email: r.Email,
        }
        human, err := idp.UpdateHuman(env.Driver, updateHuman)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if human != (idp.Human{}) {
          ok := client.Human{
            Id: human.Id,
            Username: human.Username,
            Password: human.Password,
            Name: human.Name,
            Email: human.Email,
            AllowLogin: human.AllowLogin,
            TotpRequired: human.TotpRequired,
            TotpSecret: human.TotpSecret,
            OtpRecoverCode: human.OtpRecoverCode,
            OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
            OtpDeleteCode: human.OtpDeleteCode,
            OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }
          var response client.UpdateHumansResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue
        }

        // Deny by default
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{ "username": updateHuman.Username, "name": updateHuman.Name, "email": updateHuman.Email, /* "password": updateHuman.Password, */ "allow_login":updateHuman.AllowLogin }).Debug(err.Error())
        request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_UPDATED)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func DeleteHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteHumans",
    })

    var requests []client.DeleteHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
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

    var handleRequest = func(iRequests []*utils.Request) {

      //requestedByIdentity := c.MustGet("sub").(string)

      var humans []idp.Human
      for _, request := range iRequests {
        if request.Request != nil {
          var r client.DeleteHumansRequest
          r = request.Request.(client.DeleteHumansRequest)
          humans = append(humans, idp.Human{ Identity:idp.Identity{Id:r.Id} })
        }
      }
      dbHumans, err := idp.FetchHumans(env.Driver, humans)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
      var mapHumans map[string]*idp.Human
      if ( iRequests[0] != nil ) {
        for _, human := range dbHumans {
          mapHumans[human.Id] = &human
        }
      }

      for _, request := range iRequests {
        r := request.Request.(client.DeleteHumansRequest)

        var i = mapHumans[r.Id]
        if i != nil {

          // FIXME: Use challenge system!

          challenge, err := idp.CreateDeleteChallenge(config.GetString("delete.link"), *i, 60 * 5) // Fixme configfy 60*5
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          hashedCode, err := idp.CreatePassword(challenge.Code)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          n := idp.Human{
            Identity: idp.Identity{
              Id: i.Id,
              OtpDeleteCode: hashedCode,
              OtpDeleteCodeExpire: challenge.Expire,
            },
          }
          updatedHuman, err := idp.UpdateOtpDeleteCode(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          log.WithFields(logrus.Fields{ "fixme":1, "verification_code": challenge.Code }).Debug("Delete Code. Please do not do this in production!");

          templateFile := config.GetString("delete.template.email.file")
          emailSubject := config.GetString("delete.template.email.subject")

          tplRecover, err := ioutil.ReadFile(templateFile)
          if err != nil {
            log.WithFields(logrus.Fields{ "file": templateFile }).Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

          data := DeleteTemplateData{
            Sender: sender.Name,
            Name: updatedHuman.Id,
            VerificationCode: challenge.Code,
          }

          var tpl bytes.Buffer
          if err := t.Execute(&tpl, data); err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          anEmail := idp.AnEmail{
            Subject: emailSubject,
            Body: tpl.String(),
          }

          _, err = idp.SendAnEmailToHuman(smtpConfig, updatedHuman, anEmail)
          if err != nil {
            log.WithFields(logrus.Fields{ "id": updatedHuman.Id, "file": templateFile }).Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := client.HumanRedirect{
            Id: updatedHuman.Id,
            RedirectTo: challenge.RedirectTo,
          }
          var response client.DeleteHumansResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          log.WithFields(logrus.Fields{"id":ok.Id, "redirect_to":ok.RedirectTo}).Debug("Delete Verification Requested")
          continue
        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}




