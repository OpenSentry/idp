package challenges

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type OtpChallengeRequest struct {
  OtpChallenge string `form:"otp_challenge" json:"otp_challenge,omitempty" binding:"required"`
}

type OtpChallengeResponse struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Subject      string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  IssuedAt     int64  `json:"iat" binding:"required"`
  ExpiresAt    int64  `json:"exp" binding:"required"`
  TTL          int    `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

type OtpChallengeCreateRequest struct {
  Subject     string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  TTL          int    `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

func GetCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetCollection",
    })

    var otpChallengeRequest OtpChallengeRequest

    err := c.Bind(&otpChallengeRequest)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    challenge, err := idp.FetchChallenge(env.Driver, otpChallengeRequest.OtpChallenge)
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": otpChallengeRequest.OtpChallenge,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OTP challenge"})
      c.Abort()
      return
    }

    if challenge != nil {
      c.JSON(http.StatusOK, OtpChallengeResponse{
        OtpChallenge: challenge.OtpChallenge,
        Subject: challenge.Subject,
        Audience: challenge.Audience,
        IssuedAt: challenge.IssuedAt,
        ExpiresAt: challenge.ExpiresAt,
        TTL: challenge.TTL,
        RedirectTo: challenge.RedirectTo,
        CodeType: challenge.CodeType,
        Code: challenge.Code,
      })
      return
    }

    // Deny by default
    c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func PostCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostCollection",
    })

    var input OtpChallengeCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    var hashedCode string
    if input.CodeType == "TOTP" {
      hashedCode = ""
    } else {
      hashedCode, err = idp.CreatePassword(input.Code)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Subject)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": input.Subject,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Identity"})
      c.Abort()
      return;
    }

    if identities == nil {
      log.WithFields(logrus.Fields{
        "id": input.Subject,
      }).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort()
      return;
    }

    if len(identities) > 1 {
      log.WithFields(logrus.Fields{
        "id": input.Subject,
      }).Debug("Found to many identities")
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Found to many Identity"})
      c.Abort()
      return;
    }

    identity := identities[0];

    aChallenge := idp.Challenge{
      Subject: input.Subject,
      Audience: input.Audience,
      TTL: input.TTL,
      RedirectTo: input.RedirectTo,
      CodeType: input.CodeType,
      Code: hashedCode,
    }
    challenge, err := idp.CreateChallengeForIdentity(env.Driver, identity, aChallenge)
    if err != nil {
      log.WithFields(logrus.Fields{
        "sub": input.Subject, "aud":input.Audience, "ttl": input.TTL, "redirect_to": input.RedirectTo, "code": hashedCode, "code_type": input.CodeType,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create challenge"})
      c.Abort()
      return
    }

    if challenge != nil {
      c.JSON(http.StatusOK, OtpChallengeResponse{
        OtpChallenge: challenge.OtpChallenge,
        Subject: challenge.Subject,
        Audience: challenge.Audience,
        IssuedAt: challenge.IssuedAt,
        ExpiresAt: challenge.ExpiresAt,
        TTL: challenge.TTL,
        RedirectTo: challenge.RedirectTo,
        CodeType: challenge.CodeType,
        Code: challenge.Code,
      })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "sub": input.Subject, "aud":input.Audience, "ttl": input.TTL, "redirect_to": input.RedirectTo, "code": hashedCode, "code_type": input.CodeType,
    }).Debug("No challenge created")
    c.JSON(http.StatusNotFound, gin.H{"error": "No challenge created"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}
