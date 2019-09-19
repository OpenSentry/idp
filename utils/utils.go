package utils

import (
  "bytes"
  "strings"
  "net"
  "net/http"
  "crypto/rand"
  "encoding/base64"
  "time"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/gofrs/uuid"
  hydra "github.com/charmixer/hydra/client"
)

const ERROR_INVALID_ACCESS_TOKEN = 1
const ERROR_MISSING_BEARER_TOKEN = 2
const ERROR_MISSING_REQUIRED_SCOPES = 3

type JsonError struct {
  ErrorCode int `json:"error_code" binding:"required"`
  Error     string `json:"error" binding:"required"`
}

type AuthorizationConfig struct {
  LogKey             string
  AccessTokenKey     string
  HydraConfig        *clientcredentials.Config
  HydraIntrospectUrl string
}

func GenerateRandomBytes(n int) ([]byte, error) {
  b := make([]byte, n)
  _, err := rand.Read(b)
  if err != nil {
    return nil, err
  }
  return b, nil
}

func GenerateRandomString(s int) (string, error) {
  b, err := GenerateRandomBytes(s)
  return base64.StdEncoding.EncodeToString(b), err
}

type IpData struct {
  Ip string
  Port string
}

func GetRequestIpData(r *http.Request) (IpData, error) {
  ip, port, err := net.SplitHostPort(r.RemoteAddr)
  if err != nil {
    return IpData{}, err
  }
  ret := IpData{
    Ip: ip,
    Port: port,
  }
  return ret, nil
}

func GetForwardedForIpData(r *http.Request) (IpData, error) {
  ip, port := detectForwardedForIpAndPort(r)

  ret := IpData{
    Ip: ip,
    Port: port,
  }
  return ret, nil
}

type ipRange struct {
    start net.IP
    end net.IP
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
    // strcmp type byte comparison
    if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
        return true
    }
    return false
}

var privateRanges = []ipRange{
    ipRange{
        start: net.ParseIP("10.0.0.0"),
        end:   net.ParseIP("10.255.255.255"),
    },
    ipRange{
        start: net.ParseIP("100.64.0.0"),
        end:   net.ParseIP("100.127.255.255"),
    },
    ipRange{
        start: net.ParseIP("172.16.0.0"),
        end:   net.ParseIP("172.31.255.255"),
    },
    ipRange{
        start: net.ParseIP("192.0.0.0"),
        end:   net.ParseIP("192.0.0.255"),
    },
    ipRange{
        start: net.ParseIP("192.168.0.0"),
        end:   net.ParseIP("192.168.255.255"),
    },
    ipRange{
        start: net.ParseIP("198.18.0.0"),
        end:   net.ParseIP("198.19.255.255"),
    },
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
    // my use case is only concerned with ipv4 atm
    if ipCheck := ipAddress.To4(); ipCheck != nil {
        // iterate over all our ranges
        for _, r := range privateRanges {
            // check if this ip is in a private range
            if inRange(r, ipAddress){
                return true
            }
        }
    }
    return false
}

func detectForwardedForIpAndPort(r *http.Request) (string, string) {
    for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
        addresses := strings.Split(r.Header.Get(h), ",")
        // march from right to left until we get a public address
        // that will be the address right before our proxy.
        for i := len(addresses) -1 ; i >= 0; i-- {
            ip := strings.TrimSpace(addresses[i])
            // header can contain spaces too, strip those out.
            realIP := net.ParseIP(ip)
            if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
                // bad address, go to next
                continue
            }
            return ip, ""
        }
    }
    return "", ""
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

    ipData, err := GetRequestIpData(c.Request)
    if err != nil {
      log.WithFields(appFields).WithFields(logrus.Fields{
        "func": "RequestLogger",
      }).Debug(err.Error())
    }

    forwardedForIpData, err := GetForwardedForIpData(c.Request)
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
    if len(split) == 2 || strings.EqualFold(split[0], "bearer") {

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

    log := c.MustGet(aconf.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{"func": "AuthorizationRequired"})

    // This is required to be here but should be garantueed by the authenticationRequired function.
    t, accessTokenExists := c.Get(aconf.AccessTokenKey)
    if accessTokenExists == false {
      c.AbortWithStatusJSON(http.StatusForbidden, JsonError{ErrorCode: ERROR_MISSING_BEARER_TOKEN, Error: "No access token found. Hint: Is bearer token missing?"})
      return
    }
    var accessToken *oauth2.Token = t.(*oauth2.Token)

    strRequiredScopes := strings.Join(requiredScopes, " ")
    log.WithFields(logrus.Fields{"scope": strRequiredScopes}).Debug("Checking required scopes");

    // See #3 of QTNA
    // log.WithFields(logrus.Fields{"fixme": 1, "qtna": 3}).Debug("Missing check if access token is granted the required scopes")
    hydraClient := hydra.NewHydraClient(aconf.HydraConfig)

    log.WithFields(logrus.Fields{"token": accessToken.AccessToken}).Debug("Introspecting token")

    introspectRequest := hydra.IntrospectRequest{
      Token: accessToken.AccessToken,
      Scope: strRequiredScopes, // This will make hydra check that all scopes are present else introspect.active will be false.
    }
    introspectResponse, err := hydra.IntrospectToken(aconf.HydraIntrospectUrl, hydraClient, introspectRequest)
    if err != nil {
      log.WithFields(logrus.Fields{"scope": strRequiredScopes}).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    log.Debug(introspectResponse)

    if introspectResponse.Active == true {

      // Check scopes. (is done by hydra according to doc)
      // https://www.ory.sh/docs/hydra/sdk/api#introspect-oauth2-tokens

      // See #4 of QTNA
      log.WithFields(logrus.Fields{"fixme": 1, "qtna": 4}).Debug("Missing check if the user or client giving the grants in the access token authorized to use the scopes granted")

      foundRequiredScopes := true
      if foundRequiredScopes {
        log.WithFields(logrus.Fields{"sub": introspectResponse.Sub, "scope": strRequiredScopes}).Debug("Authorized")
        c.Set("sub", introspectResponse.Sub)
        c.Next() // Authentication successful, continue.
        return;
      }
    }

    // Deny by default
    log.WithFields(logrus.Fields{"fixme": 1}).Debug("Calculate missing scopes and only log those");
    c.AbortWithStatusJSON(http.StatusForbidden, JsonError{ErrorCode: ERROR_MISSING_REQUIRED_SCOPES, Error: "Missing required scopes. Hint: Some required scopes are missing, invalid or not granted"})
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
