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

type RecoverTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func PostRecover(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecover",
    })

    var requests []client.CreateHumansRecoverRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    sender := idp.SMTPSender{
      Name: config.GetString("recover.sender.name"),
      Email: config.GetString("recover.sender.email"),
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
        r := request.Input.(client.CreateHumansRecoverRequest)

        humans, err := idp.FetchHumansById( env.Driver, []string{r.Id} )
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        log.Debug(humans)

        if len(humans) > 0 {
          human := humans[0]

          // FIXME: Use challenge system!

          challenge, err := idp.CreateRecoverChallenge(config.GetString("recover.link"), human, 60 * 5) // Fixme configfy 60*5
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
            },
            OtpRecoverCode: hashedCode,
            OtpRecoverCodeExpire: challenge.Expire,
          }
          updatedHuman, err := idp.UpdateOtpRecoverCode(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          log.WithFields(logrus.Fields{ "fixme":1, "code": challenge.Code }).Debug("Recover Code. Please do not do this in production!");

          templateFile := config.GetString("recover.template.email.file")
          emailSubject := config.GetString("recover.template.email.subject")

          tplRecover, err := ioutil.ReadFile(templateFile)
          if err != nil {
            log.WithFields(logrus.Fields{ "file": templateFile }).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

          data := RecoverTemplateData{
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

          ok := client.CreateHumansRecoverResponse{
            Id: updatedHuman.Id,
            RedirectTo: challenge.RedirectTo,
          }

          log.WithFields(logrus.Fields{"id":ok.Id, "redirect_to":ok.RedirectTo}).Debug("Recover Verification Requested")
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
