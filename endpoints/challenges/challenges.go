package challenges

import (
  "errors"
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "golang.org/x/oauth2"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/config"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/client"
  E "github.com/opensentry/idp/client/errors"

  aap "github.com/opensentry/aap/client"
  bulkyClient "github.com/charmixer/bulky/client"

  bulky "github.com/charmixer/bulky/server"
)

type ConfirmTemplateData struct {
  Challenge string
  Id string
  Code string
  Sender string
  Email string
}

func GetChallenges(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
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

      session, tx, err := idp.BeginReadTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //  if err != nil {
      //    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //    log.Debug(err.Error())
      //    return
      //  }
      //  if len(identities) > 0 {
      //    requestedBy = &identities[0]
      //  }
      // }

      for _, request := range iRequests {

        var dbChallenges []idp.Challenge
        var err error
        var ok client.ReadChallengesResponse

        if request.Input == nil {
          dbChallenges, err = idp.FetchChallenges(tx, nil)
        } else {
          r := request.Input.(client.ReadChallengesRequest)
          log = log.WithFields(logrus.Fields{"otp_challenge": r.OtpChallenge})
          dbChallenges, err = idp.FetchChallenges(tx, []idp.Challenge{ {Id: r.OtpChallenge} })
        }
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if len(dbChallenges) > 0 {
          for _, d := range dbChallenges {
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
              Data: d.Data,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PostChallenges(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostChallenges",
    })

    var requests []client.CreateChallengesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // This is required to be here but should be garantueed by the authenticationRequired function.
    t, accessTokenExists := c.Get(env.Constants.AccessTokenKey)
    if accessTokenExists == false {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Missing access token"})
      return
    }
    var token *oauth2.Token = t.(*oauth2.Token)

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //  if err != nil {
      //    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //    log.Debug(err.Error())
      //    return
      //  }
      //  if len(identities) > 0 {
      //    requestedBy = &identities[0]
      //  }
      // }

      challengeTypeRequiredScopes := map[idp.ChallengeType][]string{
        idp.ChallengeAuthenticate: []string{"idp:create:challenge.authenticate"},
        idp.ChallengeRecover: []string{"idp:create:challenge.recover"},
        idp.ChallengeDelete: []string{"idp:create:challenge.delete"},
      }

      for _, request := range iRequests {
        r := request.Input.(client.CreateChallengesRequest)

        ct := translateConfirmationTypeToChallengeType(r.ConfirmationType)
        if ct == idp.ChallengeNotSupported {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_CONFIRMATION_TYPE_INVALID)
          return
        }

        // Call judge to test if allowed to call endpoint for challenge type
        requiredScopes := challengeTypeRequiredScopes[ct]
        valid, err := judgeRequiredScope(env, c, log, token, requiredScopes...)
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        // Judgement!
        if valid == false {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewErrorResponse(request.Index, http.StatusForbidden, E.HUMAN_TOKEN_INVALID)
          return
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
        }

        var otpCode idp.ChallengeCode
        var challenge idp.Challenge
        if client.OTPType(newChallenge.CodeType) == client.TOTP {
          challenge, err = idp.CreateChallengeUsingTotp(tx, ct, newChallenge)
        } else {
          challenge, otpCode, err = idp.CreateChallengeUsingOtp(tx, ct, newChallenge)
        }
        if err == nil && challenge.Id != "" {

          if otpCode.Code != "" && r.Email != "" {

            // Sent challenge to requested email

            emailTemplate := (*env.TemplateMap)[ct]
            if emailTemplate == (app.EmailTemplate{}) {
              e := tx.Rollback()
              if e != nil {
                log.Debug(e.Error())
              }
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
              log.WithFields(logrus.Fields{ "challenge_type":ct.String() }).Debug("Email template not found")
              return
            }

            var data = ConfirmTemplateData{
              Challenge: challenge.Id,
              Sender: emailTemplate.Sender.Name,
              Id: challenge.Subject,
              Email: r.Email,
              Code: otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
            }

            smtpConfig := idp.SMTPConfig{
              Host: config.GetString("mail.smtp.host"),
              Username: config.GetString("mail.smtp.user"),
              Password: config.GetString("mail.smtp.password"),
              Sender: emailTemplate.Sender,
              SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
            }
            _, err = idp.SendEmailUsingTemplate(smtpConfig, r.Email, r.Email, emailTemplate.Subject, emailTemplate.File, data)
            if err != nil {
              e := tx.Rollback()
              if e != nil {
                log.Debug(e.Error())
              }
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
              log.Debug(err.Error())
              return
            }

          }

          confirmationType := translateChallengeTypeToConfirmationType(challenge.ChallengeType)

          request.Output = bulky.NewOkResponse(request.Index, client.CreateChallengesResponse{
            OtpChallenge: challenge.Id,
            ConfirmationType: int(confirmationType),
            Subject: challenge.Subject,
            Audience: challenge.Audience,
            IssuedAt: challenge.IssuedAt,
            ExpiresAt: challenge.ExpiresAt,
            TTL: challenge.ExpiresAt - challenge.IssuedAt,
            RedirectTo: challenge.RedirectTo,
            CodeType: challenge.CodeType,
            Code: challenge.Code,
          })
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        log.Debug(err.Error())
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func judgeRequiredScope(env *app.Environment, c *gin.Context, log *logrus.Entry, token *oauth2.Token, requiredScopes ...string) (valid bool, err error) {

  // Check that access token has required scopes
  v, exists := c.Get("scope") // scope from introspection call
  if exists == false {
    return false, errors.New("Missing scope in context")
  }
  scope := v.(string)

  // TODO: Check the access token for required scopes.
  log.Debug(scope)

  // Check that subject is granted scopes.
  v, exists = c.Get("sub") // sub from introspection call
  if exists == false {
    return false, errors.New("Missing sub in context")
  }
  sub := v.(string)


  publisherId := config.GetString("id") // Resource Server (this)

  var judgeRequests []aap.ReadEntitiesJudgeRequest
  for _, scope := range requiredScopes {
    judgeRequests = append(judgeRequests, aap.ReadEntitiesJudgeRequest{
      AccessToken: token.AccessToken,
      Publisher: publisherId,
      Scope: scope,
      Owners: []string{ sub },
    })
  }

  aapClient := aap.NewAapClient(env.AapConfig)
  url := config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.entities.judge")
  status, responses, err := aap.ReadEntitiesJudge(aapClient, url, judgeRequests)
  if err != nil {
    return false, err
  }

  if status == http.StatusOK {

    var verdict aap.ReadEntitiesJudgeResponse
    status, restErr := bulkyClient.Unmarshal(0, responses, &verdict)
    if restErr != nil {
      log.Debug(restErr)
      return false, errors.New("Unmarshal ReadEntitiesJudgeResponse failed")
    }

    if status == http.StatusOK {

      if verdict.Granted == true {
        // log.WithFields(logrus.Fields{"sub": sub, "scope": strRequiredScopes}).Debug("Authorized")
        return true, nil // Authenticated
      }

    }

  }

  // Deny by default
  return false, nil
}

func translateConfirmationTypeToChallengeType(confirmationType int) (challengeType idp.ChallengeType) {
  ct := client.ConfirmationType(confirmationType)
  switch ct {

  case client.ConfirmIdentity:
    return idp.ChallengeAuthenticate

  case client.ConfirmIdentityDeletion:
    return idp.ChallengeDelete

  case client.ConfirmIdentityRecovery:
    return idp.ChallengeRecover

  case client.ConfirmIdentityControlOfEmail:
    return idp.ChallengeEmailConfirm

  case client.ConfirmIdentityControlOfEmailDuringChange:
    return idp.ChallengeEmailChange

  default:
    return idp.ChallengeNotSupported
  }
}

func translateChallengeTypeToConfirmationType(challengeType idp.ChallengeType) (confirmationType client.ConfirmationType) {
  switch challengeType {

  case idp.ChallengeAuthenticate:
    return client.ConfirmIdentity

  case idp.ChallengeDelete:
    return client.ConfirmIdentityDeletion

  case idp.ChallengeRecover:
    return client.ConfirmIdentityRecovery

  case idp.ChallengeEmailConfirm:
    return client.ConfirmIdentityControlOfEmail

  case idp.ChallengeEmailChange:
    return client.ConfirmIdentityControlOfEmailDuringChange

  default:
    return client.ConfirmationType(0)
  }
}

