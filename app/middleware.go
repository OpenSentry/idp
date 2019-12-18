package app

import (
  "strings"
  "time"
  "net/http"
  "golang.org/x/oauth2"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/gofrs/uuid"
  oidc "github.com/coreos/go-oidc"
  "golang.org/x/oauth2/clientcredentials"

  "github.com/opensentry/idp/config"
  "github.com/opensentry/idp/utils"

  aap "github.com/opensentry/aap/client"
  bulky "github.com/charmixer/bulky/client"
)

func AccessToken(env *Environment, c *gin.Context) (*oauth2.Token) {
  t, exists := c.Get(env.Constants.ContextAccessTokenKey)
  if exists == true {
    return t.(*oauth2.Token)
  }
  return nil
}

func IdToken(env *Environment, c *gin.Context) (*oidc.IDToken) {
  t, exists := c.Get(env.Constants.ContextIdTokenKey)
  if exists == true {
    return t.(*oidc.IDToken)
  }
  return nil
}

func RequireScopes(env *Environment, requiredScopes ...string) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "RequireScopes",
    })

    if len(requiredScopes) <= 0 {
      log.Debug("'Missing required scopes'")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    _requiredScopes := FetchRequiredScopes(env, c)
    _requiredScopes = append(_requiredScopes, requiredScopes...)

    c.Set(env.Constants.ContextRequiredScopesKey, _requiredScopes)
    c.Next()
    return
  }
  return gin.HandlerFunc(fn)
}

func FetchRequiredScopes(env *Environment, c *gin.Context) (requiredScopes []string) {
  t, exists := c.Get(env.Constants.ContextRequiredScopesKey)
  if exists == true {
    return t.([]string)
  }
  return nil
}


type AuthorizationConfig struct {
  LogKey             string
  AccessTokenKey     string
  HydraConfig        *clientcredentials.Config
  HydraIntrospectUrl string
  AapConfig          *clientcredentials.Config
}

func ProcessMethodOverride(r *gin.Engine) gin.HandlerFunc {
  return func(c *gin.Context) {

    // Only need to check POST method
    if c.Request.Method != "POST" {
      return
    }

    method := c.Request.Header.Get("X-HTTP-Method-Override")
    method = strings.ToLower(method)
    method = strings.TrimSpace(method)

    // Require using method override
    if method == "" {
      c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or empty X-HTTP-Method-Override header"})
      c.Abort()
      return
    }


    if method == "post" {
      // if HandleContext is called you will make an infinite loop
      //c.Next()
      return
    }

    if method == "get" {
      c.Request.Method = "GET"
      r.HandleContext(c)
      c.Abort()
      return
    }

    if method == "put" {
      c.Request.Method = "PUT"
      r.HandleContext(c)
      c.Abort()
      return
    }

    if method == "delete" {
      c.Request.Method = "DELETE"
      r.HandleContext(c)
      c.Abort()
      return
    }

    c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported method"})
    c.Abort()
    return
  }
}


func RequestLogger(logKey string, requestIdKey string, log *logrus.Logger, appFields logrus.Fields) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Start timer
    start := time.Now()
    path := c.Request.URL.Path
    raw := c.Request.URL.RawQuery

    var requestId string = c.MustGet(requestIdKey).(string)
    requestLog := log.WithFields(appFields).WithFields(logrus.Fields{
      "request.id": requestId,
    })
    c.Set(logKey, requestLog)

    c.Next()

    // Stop timer
    stop := time.Now()
    latency := stop.Sub(start)

    ipData, err := utils.GetRequestIpData(c.Request)
    if err != nil {
      log.WithFields(appFields).WithFields(logrus.Fields{
        "func": "RequestLogger",
      }).Debug(err.Error())
    }

    forwardedForIpData, err := utils.GetForwardedForIpData(c.Request)
    if err != nil {
      log.WithFields(appFields).WithFields(logrus.Fields{
        "func": "RequestLogger",
      }).Debug(err.Error())
    }

    method := c.Request.Method
    statusCode := c.Writer.Status()
    errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

    bodySize := c.Writer.Size()

    var fullpath string = path
    if raw != "" {
      fullpath = path + "?" + raw
    }

    // if public data is requested successfully, then dont log it since its just spam when debugging
    if strings.Contains(path, "/public/") && ( statusCode == http.StatusOK || statusCode == http.StatusNotModified ) {
     return
    }

    log.WithFields(appFields).WithFields(logrus.Fields{
      "latency": latency,
      "forwarded_for.ip": forwardedForIpData.Ip,
      "forwarded_for.port": forwardedForIpData.Port,
      "ip": ipData.Ip,
      "port": ipData.Port,
      "method": method,
      "status": statusCode,
      "error": errorMessage,
      "body_size": bodySize,
      "path": fullpath,
      "request.id": requestId,
    }).Info("")
  }
  return gin.HandlerFunc(fn)
}

func AuthenticationRequired(logKey string, accessTokenKey string) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(logKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "AuthenticationRequired",
    })

    log = log.WithFields(logrus.Fields{"authorization": "bearer"})
    log.Debug("Looking for access token")
    var token *oauth2.Token
    auth := c.Request.Header.Get("Authorization")
    split := strings.SplitN(auth, " ", 2)
    if len(split) == 2 && strings.EqualFold(split[0], "bearer") {

      token = &oauth2.Token{
        AccessToken: split[1],
        TokenType: split[0],
      }

      // See #2 of QTNA
      // https://godoc.org/golang.org/x/oauth2#Token.Valid
      if token.Valid() == true {

        // See #5 of QTNA
        log.WithFields(logrus.Fields{"fixme": 1, "qtna": 5}).Debug("Missing check against token-revoked-list to check if token is revoked")

        log.Debug("Authenticated")
        c.Set(accessTokenKey, token)
        c.Next() // Authentication successful, continue.
        return;
      }

      // Deny by default
      c.JSON(http.StatusUnauthorized, JsonError{ErrorCode: ERROR_INVALID_ACCESS_TOKEN, Error: "Invalid access token."})
      c.Abort()
      return
    }

    // Deny by default
    c.JSON(http.StatusUnauthorized, JsonError{ErrorCode: ERROR_MISSING_BEARER_TOKEN, Error: "Authorization: Bearer <token> not found in request"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func AuthorizationRequired(aconf AuthorizationConfig, requiredScopes ...string) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    publisherId := config.GetString("id") // Resource Server (this)

    strRequiredScopes := strings.Join(requiredScopes, " ")

    log := c.MustGet(aconf.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{"func": "AuthorizationRequired"}).WithFields(logrus.Fields{"required_scopes": strRequiredScopes})

    // This is required to be here but should be garantueed by the authenticationRequired function.
    t, accessTokenExists := c.Get(aconf.AccessTokenKey)
    if accessTokenExists == false {
      c.AbortWithStatusJSON(http.StatusForbidden, JsonError{ErrorCode: ERROR_MISSING_BEARER_TOKEN, Error: "No access token found. Hint: Is bearer token missing?"})
      return
    }
    var token *oauth2.Token = t.(*oauth2.Token)

    var judgeRequests []aap.ReadEntitiesJudgeRequest
    for _, scope := range requiredScopes {
      judgeRequests = append(judgeRequests, aap.ReadEntitiesJudgeRequest{
        AccessToken: token.AccessToken,
        Publisher: publisherId,
        Scope: scope,
      })
    }

    aapClient := aap.NewAapClient(aconf.AapConfig)
    url := config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.entities.judge")
    status, responses, err := aap.ReadEntitiesJudge(aapClient, url, judgeRequests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if status == http.StatusForbidden {
      c.AbortWithStatus(http.StatusForbidden)
      return
    }

    if status != http.StatusOK {
      log.WithFields(logrus.Fields{"status": status}).Debug("Call aap.ReadEntitiesJudge failed");
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    var verdict aap.ReadEntitiesJudgeResponse
    status, restErr := bulky.Unmarshal(0, responses, &verdict)
    if restErr != nil {
      log.Debug(restErr)
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if status == http.StatusOK {

      if verdict.Granted == true {

        log.WithFields(logrus.Fields{"id": verdict.Identity, "scopes": verdict.Scope}).Debug("Authorized")

        c.Set("sub", verdict.Identity)
        c.Set("scope", verdict.Scope)
        c.Next() // Authentication successful, continue.
        return
      }

      log.WithFields(logrus.Fields{"id": verdict.Identity, "scopes": verdict.Scope}).Debug("Forbidden")
      c.AbortWithStatusJSON(http.StatusForbidden, JsonError{})
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"status": status}).Debug("Unmarshal ReadEntitiesJudgeResponse failed");
    c.AbortWithStatus(http.StatusInternalServerError)
    return
  }
  return gin.HandlerFunc(fn)
}


func RequestId() gin.HandlerFunc {
  return func(c *gin.Context) {
    // Check for incoming header, use it if exists
    requestID := c.Request.Header.Get("X-Request-Id")

    // Create request id with UUID4
    if requestID == "" {
      uuid4, _ := uuid.NewV4()
      requestID = uuid4.String()
    }

    // Expose it for use in the application
    c.Set("RequestId", requestID)

    // Set X-Request-Id header
    c.Writer.Header().Set("X-Request-Id", requestID)
    c.Next()
  }
}
