package challenges

import (
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type OtpChallengeRequest struct {
  OtpChallenge string `form:"otp_challenge" json:"otp_challenge" binding:"required"`
}

type OtpChallengeResponse struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Subject      string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  IssuedAt     int64  `json:"iat" binding:"required"`
  ExpiresAt    int64  `json:"exp" binding:"required"`
  TTL          int64  `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

type OtpChallengeCreateRequest struct {
  Subject     string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  TTL          int64  `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

func GetChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetCollection",
    })

    var otpChallengeRequest OtpChallengeRequest

    err := c.Bind(&otpChallengeRequest)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    challenges, err := idp.FetchChallengesById(env.Driver, []string{otpChallengeRequest.OtpChallenge})
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": otpChallengeRequest.OtpChallenge,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if challenges != nil {
      challenge := challenges[0]

      c.JSON(http.StatusOK, OtpChallengeResponse{
        OtpChallenge: challenge.Id,
        Subject: challenge.Subject,
        Audience: challenge.Audience,
        IssuedAt: challenge.IssuedAt,
        ExpiresAt: challenge.ExpiresAt,
        TTL: challenge.ExpiresAt - challenge.IssuedAt,
        RedirectTo: challenge.RedirectTo,
        CodeType: challenge.CodeType,
        Code: challenge.Code,
      })
      return
    }

    // Deny by default
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
  }
  return gin.HandlerFunc(fn)
}

func PostChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostCollection",
    })

    var input OtpChallengeCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var hashedCode string
    if input.CodeType == "TOTP" {
      hashedCode = ""
    } else {
      hashedCode, err = idp.CreatePassword(input.Code)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
    }

    identities, err := idp.FetchHumansById(env.Driver, []string{input.Subject})
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": input.Subject,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if identities == nil {
      log.WithFields(logrus.Fields{
        "id": input.Subject,
      }).Debug("Identity not found")
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }
    identity := identities[0]

    newChallenge := idp.Challenge{
      JwtRegisteredClaims: idp.JwtRegisteredClaims{
        Subject: identity.Id,
        Audience: input.Audience,
        ExpiresAt: time.Now().Unix() + input.TTL,
      },
      RedirectTo: input.RedirectTo,
      CodeType: input.CodeType,
      Code: hashedCode,
    }
    challenge, _, err := idp.CreateChallenge(env.Driver, newChallenge, identity)
    if err != nil {
      log.WithFields(logrus.Fields{
        "sub": input.Subject, "aud":input.Audience, "ttl": input.TTL, "redirect_to": input.RedirectTo, "code": hashedCode, "code_type": input.CodeType,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    c.JSON(http.StatusOK, OtpChallengeResponse{
      OtpChallenge: challenge.Id,
      Subject: challenge.Subject,
      Audience: challenge.Audience,
      IssuedAt: challenge.IssuedAt,
      ExpiresAt: challenge.ExpiresAt,
      TTL: challenge.ExpiresAt - challenge.IssuedAt,
      RedirectTo: challenge.RedirectTo,
      CodeType: challenge.CodeType,
      Code: challenge.Code,
    })
    return

    // Deny by default
    //log.WithFields(logrus.Fields{
    //  "sub": input.Subject, "aud":input.Audience, "ttl": input.TTL, "redirect_to": input.RedirectTo, "code": hashedCode, "code_type": input.CodeType,
    //}).Debug("Challenge not created")
    //c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not created"})
  }
  return gin.HandlerFunc(fn)
}
