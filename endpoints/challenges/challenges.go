package challenges

import (
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
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
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request){
      var challenges []idp.Challenge

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadChallengesRequest
          r = request.Request.(client.ReadChallengesRequest)

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
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      for _, request := range iRequests {
        var r client.ReadChallengesRequest
        if request.Request != nil {
          r = request.Request.(client.ReadChallengesRequest)
        }

        var ok []client.Challenge
        for _, d := range dbChallenges {
          if request.Request != nil && d.Id != r.OtpChallenge {
            continue
          }

          ok = append(ok, client.Challenge{
            OtpChallenge: d.Id,
          })
        }

        var response client.ReadChallengesResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = ok
        request.Response = response
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})

    c.JSON(http.StatusOK, responses)

    // Deny by default
    //c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
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

    var handleRequest = func(iRequests []*utils.Request) {

      requestedByIdentity := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.CreateChallengesRequest)

        var hashedCode string
        if r.CodeType == "TOTP" {
          hashedCode = ""
        } else {
          hashedCode, err = idp.CreatePassword(r.Code)
          if err != nil {
            request.Response = utils.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
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
          request.Response = utils.NewInternalErrorResponse(request.Index)
          log.WithFields(logrus.Fields{
            "sub": challenge.Subject, "aud":challenge.Audience, "exp": challenge.ExpiresAt, "redirect_to": challenge.RedirectTo, "code": hashedCode, "code_type": challenge.CodeType,
          }).Debug(err.Error())
          continue
        }

        ok := client.Challenge{
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

        response := client.CreateChallengesResponse{Ok: ok}
        response.Index = request.Index
        response.Status = http.StatusOK
        request.Response = response
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})

    c.JSON(http.StatusOK, responses)

    // Deny by default
    //log.WithFields(logrus.Fields{
    //  "sub": input.Subject, "aud":input.Audience, "ttl": input.TTL, "redirect_to": input.RedirectTo, "code": hashedCode, "code_type": input.CodeType,
    //}).Debug("Challenge not created")
    //c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not created"})
  }
  return gin.HandlerFunc(fn)
}
