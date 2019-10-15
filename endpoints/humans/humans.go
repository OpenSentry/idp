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
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {

        var dbHumans []idp.Human
        var err error
        var ok client.ReadHumansResponse

        if request.Input == nil {

          // Fetch all, that the token is allowed to.
          dbHumans, err = idp.FetchHumansAll(env.Driver)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        } else {

          r := request.Input.(client.ReadHumansRequest)
          if r.Id != "" {
            dbHumans, err = idp.FetchHumansById(env.Driver, []string{r.Id})
          } else if r.Email != "" {
            dbHumans, err = idp.FetchHumansByEmail(env.Driver, []string{r.Email})
          } else if r.Username != "" {
            dbHumans, err = idp.FetchHumansByUsername(env.Driver, []string{r.Username})
          }

          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        }

        if len(dbHumans) > 0 {
          for _, i := range dbHumans {
            ok = append(ok, client.Human{
              Id:                   i.Id,
              Username:             i.Username,
              //Password:             i.Password,
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
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue;
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansRequest)

        if env.BannedUsernames[r.Username] == true {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_BANNED)
          continue
        }

        // Sanity check. Username must be unique
        if r.Username != "" {
          humansFoundByUsername, err := idp.FetchHumansByUsername(env.Driver, []string{r.Username})
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
          if len(humansFoundByUsername) > 0 {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_EXISTS)
            continue
          }
        }

        hashedPassword, err := idp.CreatePassword(r.Password) // @SecurityRisk: Please _NEVER_ log the cleartext password
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        newHuman := idp.Human{
          Identity: idp.Identity{ Id: r.Id },
          Username: r.Username,
          Name: r.Name,
          Password: hashedPassword,
          AllowLogin: true,
          EmailConfirmedAt: r.EmailConfirmedAt,
        }
        human, err := idp.CreateHumanFromInvite(env.Driver, newHuman)
        if err != nil {
          // @SecurityRisk: Please _NEVER_ log the hashed password
          log.WithFields(logrus.Fields{ "username": newHuman.Username, "name": newHuman.Name, "email": newHuman.Email, /* "password": newHuman.Password, */ "allow_login":newHuman.AllowLogin, "email_confirmed_at":newHuman.EmailConfirmedAt }).Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if human != (idp.Human{}) {
          ok := client.CreateHumansResponse{
              Id: human.Id,
              Username: human.Username,
              Password: human.Password,
              Name: human.Name,
              Email: human.Email,
              EmailConfirmedAt: human.EmailConfirmedAt,
              AllowLogin: human.AllowLogin,
              TotpRequired: human.TotpRequired,
              TotpSecret: human.TotpSecret,
              OtpRecoverCode: human.OtpRecoverCode,
              OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
              OtpDeleteCode: human.OtpDeleteCode,
              OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          idp.EmitEventHumanCreated(env.Nats, human)
          continue
        }

        // Deny by default
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{ "username": newHuman.Username, "name": newHuman.Name, "email": newHuman.Email, /* "password": newHuman.Password, */ "allow_login":newHuman.AllowLogin, "email_confirmed_at":newHuman.EmailConfirmedAt }).Debug(err.Error())
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_CREATED)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
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

    var handleRequests = func(iRequests []*bulky.Request) {

      //requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansRequest)

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
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if human != (idp.Human{}) {
          ok := client.UpdateHumansResponse{
            Id: human.Id,
            Username: human.Username,
            //Password: human.Password,
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
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{ "username": updateHuman.Username, "name": updateHuman.Name, "email": updateHuman.Email, /* "password": updateHuman.Password, */ "allow_login":updateHuman.AllowLogin }).Debug(err.Error())
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_UPDATED)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{})
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

    var handleRequests = func(iRequests []*bulky.Request) {

      //requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.DeleteHumansRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        humans, err := idp.FetchHumansById( env.Driver, []string{r.Id} )
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if humans == nil {
          log.WithFields(logrus.Fields{"id": r.Id}).Debug("Human not found")
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        human := humans[0]

        if human != (idp.Human{}) {

          // FIXME: Use challenge system!

          challenge, err := idp.CreateDeleteChallenge(config.GetString("delete.link"), human, 60 * 5) // Fixme configfy 60*5
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          hashedCode, err := idp.CreatePassword(challenge.Code)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          n := idp.Human{
            Identity: idp.Identity{
              Id: human.Id,
              OtpDeleteCode: hashedCode,
              OtpDeleteCodeExpire: challenge.Expire,
            },
          }
          updatedHuman, err := idp.UpdateOtpDeleteCode(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          log.WithFields(logrus.Fields{ "fixme":1, "verification_code": challenge.Code }).Debug("Delete Code. Please do not do this in production!");

          templateFile := config.GetString("delete.template.email.file")
          emailSubject := config.GetString("delete.template.email.subject")

          tplRecover, err := ioutil.ReadFile(templateFile)
          if err != nil {
            log.WithFields(logrus.Fields{ "file": templateFile }).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
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
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          anEmail := idp.AnEmail{
            Subject: emailSubject,
            Body: tpl.String(),
          }

          _, err = idp.SendAnEmailToHuman(smtpConfig, updatedHuman, anEmail)
          if err != nil {
            log.WithFields(logrus.Fields{ "id": updatedHuman.Id, "file": templateFile }).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := client.DeleteHumansResponse{
            Id: updatedHuman.Id,
            RedirectTo: challenge.RedirectTo,
          }

          log.WithFields(logrus.Fields{"id":ok.Id, "redirect_to":ok.RedirectTo}).Debug("Delete Verification Requested")
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}




