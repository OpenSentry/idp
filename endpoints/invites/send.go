package invites

import (
  "text/template"
  "io/ioutil"
  "bytes"
  "net/url"

  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

type InviteTemplateData struct {
  Id string
  InvitedBy string
  Email string
  InvitationUrl string
  IdentityProvider string
}

func PutInvitesSend(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutInvitesSend",
    })

    var requests []client.CreateInvitesSendRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
      log.WithFields(logrus.Fields{ "file": emailTemplateFile }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    t := template.Must(template.New(emailTemplateFile).Parse(string(tplEmail)))


    var handleRequests = func(iRequests []*bulky.Request) {

      //sendByIdentityId := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesSendRequest)

        dbInvite, err := idp.FetchInvitesById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if len(dbInvite) > 0 {
          invite := dbInvite[0]

          u, err := url.Parse( config.GetString("invite.url") )
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          q := u.Query()
          q.Add("id", invite.Id)
          u.RawQuery = q.Encode()

          data := InviteTemplateData{
            Id: invite.Id,
            InvitedBy: invite.InvitedBy.Name,
            Email: invite.SentTo.Email,
            InvitationUrl: u.String(),
            IdentityProvider: config.GetString("provider.name"),
          }

          var tpl bytes.Buffer
          if err := t.Execute(&tpl, data); err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          mail := idp.AnEmail{
            Subject: emailSubject,
            Body: tpl.String(),
          }

          _, err = idp.SendAnEmailToAnonymous(smtpConfig, invite.SentTo.Email, invite.SentTo.Email, mail)
          if err != nil {
            log.WithFields(logrus.Fields{ "email": invite.SentTo.Email, "file": emailTemplateFile }).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := client.CreateInvitesResponse{
            Id: invite.Id,
            IssuedAt: invite.IssuedAt,
            ExpiresAt: invite.ExpiresAt,
            Email: invite.SentTo.Email,
            Invited: invite.Invited.Id,
            HintUsername: invite.HintUsername,
            InvitedBy: invite.InvitedBy.Id,
          }

          log.WithFields(logrus.Fields{ "id": ok.Id, }).Debug("Invite sent")
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        log.WithFields(logrus.Fields{ "id":r.Id }).Debug(err.Error())
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }

    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
