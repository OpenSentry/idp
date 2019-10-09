package challenges

import (
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"

  bulky "github.com/charmixer/bulky/server"
)

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

      requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.CreateChallengesRequest)

        var hashedCode string
        if r.CodeType == "TOTP" {
          hashedCode = ""
        } else {
          hashedCode, err = idp.CreatePassword(r.Code)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
        }

        challenge := idp.Challenge{
          JwtRegisteredClaims: idp.JwtRegisteredClaims{
            Subject: requestedByIdentity,
            Audience: r.Audience,
            ExpiresAt: time.Now().Unix() + r.TTL,
          },
          RedirectTo: r.RedirectTo,
          CodeType: r.CodeType,
          Code: hashedCode,
        }

        rChallenge, _, err := idp.CreateChallenge(env.Driver, challenge, requestedByIdentity)
        if err != nil {
          log.WithFields(logrus.Fields{
            "sub": challenge.Subject, "aud":challenge.Audience, "exp": challenge.ExpiresAt, "redirect_to": challenge.RedirectTo, "code": hashedCode, "code_type": challenge.CodeType,
          }).Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        ok := client.CreateChallengesResponse{
          OtpChallenge: rChallenge.Id,
          Subject: rChallenge.Subject,
          Audience: rChallenge.Audience,
          IssuedAt: rChallenge.IssuedAt,
          ExpiresAt: rChallenge.ExpiresAt,
          TTL: rChallenge.ExpiresAt - rChallenge.IssuedAt,
          RedirectTo: rChallenge.RedirectTo,
          CodeType: rChallenge.CodeType,
          Code: rChallenge.Code,
        }
        request.Output = bulky.NewOkResponse(request.Index, ok)
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
