package challenges

import (
  "time"
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

  bulky "github.com/charmixer/bulky/server"
)

type ConfirmTemplateData struct {
  Challenge string
  Id string
  Code string
  Sender string
  Email string
}

func GetChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetChallenges",
    })

    var requests []client.ReadChallengesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {
      var challenges []idp.Challenge

      for _, request := range iRequests {
        if request.Input != nil {
          var r client.ReadChallengesRequest
          r = request.Input.(client.ReadChallengesRequest)

          // Translate from rest model to db
          v := idp.Challenge{
            Id: r.OtpChallenge,
          }
          challenges = append(challenges, v)
        }
      }

      dbChallenges, err := idp.FetchChallenges(env.Driver, challenges)
      if err != nil {
        log.Debug(err.Error())
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        return
      }

      for _, request := range iRequests {
        var r client.ReadChallengesRequest
        if request.Input != nil {
          r = request.Input.(client.ReadChallengesRequest)
        }

        var ok client.ReadChallengesResponse
        for _, d := range dbChallenges {
          if request.Input != nil && d.Id != r.OtpChallenge {
            continue
          }

          // Translate from db model to rest model
          ok = append(ok, client.Challenge{
            OtpChallenge: d.Id,
            Subject: d.Subject,
            Audience: d.Audience,
            IssuedAt: d.IssuedAt,
            ExpiresAt: d.ExpiresAt,
            TTL: d.ExpiresAt - d.IssuedAt,
            RedirectTo: d.RedirectTo,
            CodeType: d.CodeType,
            VerifiedAt: d.VerifiedAt,
          })
        }

        request.Output = bulky.NewOkResponse(request.Index, ok)
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PostChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostChallenges",
    })

    var requests []client.CreateChallengesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateChallengesRequest)

        var challenge *idp.Challenge
        if client.OTPType(r.CodeType) == client.TOTP {

          challenge, err = CreateChallengeForTOTP(env, r)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        } else {

          challenge, err = CreateChallengeForOTP(env, r)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        }

        ok := client.CreateChallengesResponse{
          OtpChallenge: challenge.Id,
          Subject: challenge.Subject,
          Audience: challenge.Audience,
          IssuedAt: challenge.IssuedAt,
          ExpiresAt: challenge.ExpiresAt,
          TTL: challenge.ExpiresAt - challenge.IssuedAt,
          RedirectTo: challenge.RedirectTo,
          CodeType: challenge.CodeType,
          Code: challenge.Code,
        }
        request.Output = bulky.NewOkResponse(request.Index, ok)
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func CreateChallengeForOTP(env *environment.State, r client.CreateChallengesRequest) (*idp.Challenge, error) {

  otpCode, err := idp.CreateChallengeCode()
  if err != nil {
    return nil, err
  }

  hashedCode, err := idp.CreatePassword(otpCode.Code)
  if err != nil {
    return nil, err
  }

  newChallenge := idp.Challenge{
    JwtRegisteredClaims: idp.JwtRegisteredClaims{
      Subject: r.Subject,
      Issuer: config.GetString("idp.public.issuer"),
      Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
      ExpiresAt: time.Now().Unix() + r.TTL,
    },
    RedirectTo: r.RedirectTo,
    CodeType: r.CodeType,
    Code: hashedCode,
  }
  challenge, err := idp.CreateChallenge(env.Driver, newChallenge)
  if err != nil {
    return nil, err
  }

  if r.SentTo != "" {

    var templateFile string
    var emailSubject string
    var sender idp.SMTPSender

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    switch (r.Template) {
    case client.ConfirmEmail:
      sender = idp.SMTPSender{ Name: config.GetString("emailconfirm.sender.name"), Email: config.GetString("emailconfirm.sender.email") }
      templateFile = config.GetString("emailconfirm.template.email.file")
      emailSubject = config.GetString("emailconfirm.template.email.subject")
    default:
      sender = idp.SMTPSender{ Name: config.GetString("otp.sender.name"), Email: config.GetString("otp.sender.email") }
      templateFile = config.GetString("otp.template.email.file")
      emailSubject = config.GetString("otp.template.email.subject")
    }

    tplRecover, err := ioutil.ReadFile(templateFile)
    if err != nil {
      return nil, err
    }

    t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

    var tpl bytes.Buffer
    var data = ConfirmTemplateData{
      Challenge: challenge.Id,
      Sender: sender.Name,
      Id: challenge.Subject,
      Email: r.SentTo,
      Code: otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
    }
    if err := t.Execute(&tpl, data); err != nil {
      return nil, err
    }

    anEmail := idp.AnEmail{ Subject:emailSubject, Body:tpl.String() }

    _, err = idp.SendAnEmailToAnonymous(smtpConfig, r.SentTo, r.SentTo, anEmail)
    if err != nil {
      return nil, err
    }

  }

  return &challenge, nil
}

func CreateChallengeForTOTP(env *environment.State, r client.CreateChallengesRequest) (*idp.Challenge, error) {
  newChallenge := idp.Challenge{
    JwtRegisteredClaims: idp.JwtRegisteredClaims{
      Subject: r.Subject,
      Issuer: config.GetString("idp.public.issuer"),
      Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
      ExpiresAt: time.Now().Unix() + r.TTL,
    },
    RedirectTo: r.RedirectTo,
    CodeType: r.CodeType,
    Code: "",
  }
  challenge, err := idp.CreateChallenge(env.Driver, newChallenge)
  if err != nil {
    return nil, err
  }
  return &challenge, nil
}

