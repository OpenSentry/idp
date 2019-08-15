package main

import (
  "strings"
  "net/http"
  "net/url"
  "os"
  "time"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"
  "github.com/sirupsen/logrus"
  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"
  "github.com/atarantini/ginrequestid"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/identities"
  "github.com/pborman/getopt"
)

const app = "idpapi"

var (
  logDebug int // Set to 1 to enable debug
  logFormat string // Current only supports default and json

  log *logrus.Logger

  appFields logrus.Fields
)

func init() {
  log = logrus.New();

  err := config.InitConfigurations()
  if err != nil {
    log.Panic(err.Error())
    return
  }

  logDebug = config.GetInt("log.debug")
  logFormat = config.GetString("log.format")

  // We only have 2 log levels. Things developers care about (debug) and things the user of the app cares about (info)
  log = logrus.New();
  if logDebug == 1 {
    log.SetLevel(logrus.DebugLevel)
  } else {
    log.SetLevel(logrus.InfoLevel)
  }
  if logFormat == "json" {
    log.SetFormatter(&logrus.JSONFormatter{})
  }

  appFields = logrus.Fields{
    "appname": app,
    "log.debug": logDebug,
    "log.format": logFormat,
  }
}

func main() {
  // https://medium.com/neo4j/neo4j-go-driver-is-out-fbb4ba5b3a30
  // Each driver instance is thread-safe and holds a pool of connections that can be re-used over time. If you don’t have a good reason to do otherwise, a typical application should have a single driver instance throughout its lifetime.
  log.WithFields(appFields).Debug("Fixme Neo4j loggning should go trough logrus so it does not differ in output from rest of the app")
  driver, err := neo4j.NewDriver(config.GetString("neo4j.uri"), neo4j.BasicAuth(config.GetString("neo4j.username"), config.GetString("neo4j.password"), ""), func(config *neo4j.Config) {
    /*if logDebug == 1 {
      config.Log = neo4j.ConsoleLogger(neo4j.DEBUG)
    } else {
      config.Log = neo4j.ConsoleLogger(neo4j.INFO)
    }*/
  });
  if err != nil {
    log.WithFields(appFields).WithFields(logrus.Fields{"component": "Storage"}).Debug("neo4j.NewDriver" + err.Error())
    return
  }
  defer driver.Close()

  provider, err := oidc.NewProvider(context.Background(), config.GetString("hydra.public.url") + "/")
  if err != nil {
    log.WithFields(appFields).WithFields(logrus.Fields{"component": "Hydra Provider"}).Debug("oidc.NewProvider" + err.Error())
    return
  }

  // Setup the hydra client idpapi is going to use (oauth2 client credentials)
  // NOTE: We store the hydraConfig also as we are going to need it to let idpapi app start the Oauth2 Authorization code flow.
  hydraConfig := &clientcredentials.Config{
    ClientID:     config.GetString("oauth2.client.id"),
    ClientSecret: config.GetString("oauth2.client.secret"),
    TokenURL:     provider.Endpoint().TokenURL,
    Scopes:       config.GetStringSlice("oauth2.scopes.required"),
    EndpointParams: url.Values{"audience": {"hydra"}},
    AuthStyle: 2, // https://godoc.org/golang.org/x/oauth2#AuthStyle
  }

  // Setup app state variables. Can be used in handler functions by doing closures see exchangeAuthorizationCodeCallback
  env := &environment.State{
    Provider: provider,
    HydraConfig: hydraConfig,
    Driver: driver,
  }

  //optServe := getopt.BoolLong("serve", 0, "Serve application")
  optHelp := getopt.BoolLong("help", 0, "Help")
  getopt.Parse()

  if *optHelp {
    getopt.Usage()
    os.Exit(0)
  }

  //if *optServe {
    serve(env)
  /*} else {
    getopt.Usage()
    os.Exit(0)
  }*/

}

func serve(env *environment.State) {
  // Setup routes to use, this defines log for debug log
  routes := map[string]environment.Route{
    "/identities":              environment.Route{URL: "/identities",              LogId: "idpapi://identities"},
    "/identities/authenticate": environment.Route{URL: "/identities/authenticate", LogId: "idpui://identities/authenticate"},
    "/identities/password":     environment.Route{URL: "/identities/password",     LogId: "idpapi://identities/password"},
    "/identities/logout":       environment.Route{URL: "/identities/logout",       LogId: "idpui://identities/logout"},
    "/identities/revoke":       environment.Route{URL: "/identities/revoke",       LogId: "idpui://identities/revoke"},
    "/identities/recover":      environment.Route{URL: "/identities/recover",      LogId: "idpui://identities/recover"},
  }

  r := gin.New() // Clean gin to take control with logging.
  r.Use(gin.Recovery())

  r.Use(ginrequestid.RequestId())
  r.Use(RequestLogger(env))

  // ## QTNA - Questions that need answering before granting access to a protected resource
  // 1. Is the user or client authenticated? Answered by the process of obtaining an access token.
  // 2. Is the access token expired?
  // 3. Is the access token granted the required scopes?
  // 4. Is the user or client giving the grants in the access token authorized to operate the scopes granted?
  // 5. Is the access token revoked?

  // All requests need to be authenticated.
  r.Use(authenticationRequired())

  r.GET(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpapi.identities.get"), identities.GetCollection(env, routes["/identities"]))
  r.POST(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpapi.identities.post"), identities.PostCollection(env, routes["/identities"]))
  r.PUT(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpapi.identities.put"), identities.PutCollection(env, routes["/identities"]))

  r.POST(routes["/identities/authenticate"].URL, authorizationRequired(routes["/identities/authenticate"], "idpapi.authenticate"), identities.PostAuthenticate(env, routes["/identities/authenticate"]))
  r.POST(routes["/identities/password"].URL, authorizationRequired(routes["/identities/password"], "idpapi.authenticate"), identities.PostPassword(env, routes["/identities/password"]))

  r.POST(routes["/identities/logout"].URL, authorizationRequired(routes["/identities/logout"], "idpapi.logout"), identities.PostLogout(env, routes["/identities/logout"]))
  r.POST(routes["/identities/revoke"].URL, authorizationRequired(routes["/identities/revoke"], "idpapi.revoke"), identities.PostRevoke(env, routes["/identities/revoke"]))
  r.POST(routes["/identities/recover"].URL, authorizationRequired(routes["/identities/recover"], "idpapi.recover"), identities.PostRevoke(env, routes["/identities/recover"]))

  r.RunTLS(":" + config.GetString("serve.public.port"), config.GetString("serve.tls.cert.path"), config.GetString("serve.tls.key.path"))
}

func RequestLogger(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Start timer
    start := time.Now()
    path := c.Request.URL.Path
    raw := c.Request.URL.RawQuery

    var requestId string = c.MustGet(environment.RequestIdKey).(string)
    requestLog := log.WithFields(appFields).WithFields(logrus.Fields{
      "request.id": requestId,
    })
    c.Set(environment.LogKey, requestLog)

		c.Next()

		// Stop timer
		stop := time.Now()
		latency := stop.Sub(start)

    ipData, err := getRequestIpData(c.Request)
    if err != nil {
      log.WithFields(appFields).WithFields(logrus.Fields{
        "func": "RequestLogger",
      }).Debug(err.Error())
    }

    forwardedForIpData, err := getForwardedForIpData(c.Request)
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
    }).Info("")
  }
  return gin.HandlerFunc(fn)
}

func authenticationRequired() gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "authenticationRequired",
    })

    log.Debug("Checking Authorization: Bearer <token> in request")

    var token *oauth2.Token
    auth := c.Request.Header.Get("Authorization")
    split := strings.SplitN(auth, " ", 2)
    if len(split) == 2 || strings.EqualFold(split[0], "bearer") {

      log.Debug("Authorization: Bearer <token> found for request")

      token = &oauth2.Token{
        AccessToken: split[1],
        TokenType: split[0],
      }

      // See #2 of QTNA
      // https://godoc.org/golang.org/x/oauth2#Token.Valid
      if token.Valid() == true {
        log.Debug("Valid access token")

        // See #5 of QTNA
        log.WithFields(logrus.Fields{
          "fixme": 1,
        }).Debug("Missing check against token-revoked-list to check if token is revoked")

        c.Set(environment.AccessTokenKey, token)
        c.Next() // Authentication successful, continue.
        return;
      }

      // Deny by default
      log.Debug("Invalid Access token")
      c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token."})
      c.Abort()
      return
    }

    // Deny by default
    log.Debug("Missing access token")
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization: Bearer <token> not found in request."})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func authorizationRequired(route environment.Route, requiredScopes ...string) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "authorizationRequired",
    })

    log.Debug("Checking Authorization: Bearer <token> in request")

    _ /*accessToken*/, accessTokenExists := c.Get(environment.AccessTokenKey)
    if accessTokenExists == false {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "No access token found. Hint: Is bearer token missing?"})
      c.Abort()
      return
    }

    strRequiredScopes := strings.Join(requiredScopes, ",")
    log.Debug("Required scopes: " + strRequiredScopes);

    // See #3 of QTNA
    log.WithFields(logrus.Fields{
      "fixme": 1,
    }).Debug("Missing check if access token is granted the required scopes")

    // See #4 of QTNA
    log.WithFields(logrus.Fields{
      "fixme": 1,
    }).Debug("Missing check if the user or client giving the grants in the access token authorized to use the scopes granted")

    foundRequiredScopes := true
    if foundRequiredScopes {
      log.Debug("Valid scopes: " + strRequiredScopes)
      c.Next() // Authentication successful, continue.
      return;
    }

    // Deny by default
    log.Debug("Invalid scopes: " + strRequiredScopes + " Hint: Some required scopes are missing, invalid or not granted")
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid scopes. Hint: Some required scopes are missing, invalid or not granted"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}
