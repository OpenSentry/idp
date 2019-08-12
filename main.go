package main

import (
  "strings"
  "net/http"
  "net/url"
  "os"

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

const app = "idpbe"

func init() {
  logrus.SetFormatter(&logrus.JSONFormatter{})
  config.InitConfigurations()
}

func main() {

  // The app log
  appFields := logrus.Fields{
    "appname": app,
    "func": "main",
  }

  // https://medium.com/neo4j/neo4j-go-driver-is-out-fbb4ba5b3a30
  // Each driver instance is thread-safe and holds a pool of connections that can be re-used over time. If you don’t have a good reason to do otherwise, a typical application should have a single driver instance throughout its lifetime.
  driver, err := neo4j.NewDriver(config.GetString("neo4j.uri"), neo4j.BasicAuth(config.GetString("neo4j.username"), config.GetString("neo4j.password"), ""), func(config *neo4j.Config) {
    config.Log = neo4j.ConsoleLogger(neo4j.DEBUG)
  });
  if err != nil {
    logrus.WithFields(appFields).WithFields(logrus.Fields{"component": "Storage"}).Fatal("neo4j.NewDriver" + err.Error())
    return
  }
  defer driver.Close()

  provider, err := oidc.NewProvider(context.Background(), config.GetString("hydra.public.url") + "/")
  if err != nil {
    logrus.WithFields(appFields).WithFields(logrus.Fields{"component": "Hydra Provider"}).Fatal("oidc.NewProvider" + err.Error())
    return
  }

  // Setup the hydra client idpbe is going to use (oauth2 client credentials)
  // NOTE: We store the hydraConfig also as we are going to need it to let idpbe app start the Oauth2 Authorization code flow.
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
    AppName: app,
    Provider: provider,
    HydraConfig: hydraConfig,
    Driver: driver,
  }

  optServe := getopt.BoolLong("serve", 0, "Serve application")
  optHelp := getopt.BoolLong("help", 0, "Help")
  getopt.Parse()

  if *optHelp {
    getopt.Usage()
    os.Exit(0)
  }

  if *optServe {
    serve(env)
  } else {
    getopt.Usage()
    os.Exit(0)
  }

}

func serve(env *environment.State) {
  // Setup routes to use, this defines log for debug log
  routes := map[string]environment.Route{
    "/identities": environment.Route{
       URL: "/identities",
       LogId: "idpbe://identities",
    },
    "/identities/authenticate": environment.Route{
      URL: "/identities/authenticate",
      LogId: "idpfe://identities/authenticate",
    },
    "/identities/logout": environment.Route{
      URL: "/identities/logout",
      LogId: "idpfe://identities/logout",
    },
    "/identities/revoke": environment.Route{
      URL: "/identities/revoke",
      LogId: "idpfe://identities/revoke",
    },
    "/identities/recover": environment.Route{
      URL: "/identities/recover",
      LogId: "idpfe://identities/recover",
    },
  }

  r := gin.Default()
  r.Use(ginrequestid.RequestId())
  r.Use(logger(env))

  // ## QTNA - Questions that need answering before granting access to a protected resource
  // 1. Is the user or client authenticated? Answered by the process of obtaining an access token.
  // 2. Is the access token expired?
  // 3. Is the access token granted the required scopes?
  // 4. Is the user or client giving the grants in the access token authorized to operate the scopes granted?
  // 5. Is the access token revoked?

  // All requests need to be authenticated.
  r.Use(authenticationRequired())

  r.GET(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpbe.identities.get"), identities.GetCollection(env, routes["/identities"]))
  r.POST(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpbe.identities.post"), identities.PostCollection(env, routes["/identities"]))
  r.PUT(routes["/identities"].URL, authorizationRequired(routes["/identities"], "idpbe.identities.put"), identities.PutCollection(env, routes["/identities"]))
  r.POST(routes["/identities/authenticate"].URL, authorizationRequired(routes["/identities/authenticate"], "idpbe.authenticate"), identities.PostAuthenticate(env, routes["/identities/authenticate"]))
  r.POST(routes["/identities/logout"].URL, authorizationRequired(routes["/identities/logout"], "idpbe.logout"), identities.PostLogout(env, routes["/identities/logout"]))
  r.POST(routes["/identities/revoke"].URL, authorizationRequired(routes["/identities/revoke"], "idpbe.revoke"), identities.PostRevoke(env, routes["/identities/revoke"]))
  r.POST(routes["/identities/recover"].URL, authorizationRequired(routes["/identities/recover"], "idpbe.recover"), identities.PostRevoke(env, routes["/identities/recover"]))

  r.RunTLS(":" + config.GetString("serve.public.port"), config.GetString("serve.tls.cert.path"), config.GetString("serve.tls.key.path"))
}

func logger(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    var requestId string = c.MustGet(environment.RequestIdKey).(string)
    logger := logrus.New() // Use this to direct request log somewhere else than app log
    logger.SetFormatter(&logrus.JSONFormatter{})
    requestLog := logger.WithFields(logrus.Fields{
      "appname": env.AppName,
      "requestid": requestId,
    })
    c.Set(environment.LogKey, requestLog)
    c.Next()
  }
  return gin.HandlerFunc(fn)
}

func authenticationRequired() gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "main.authenticationRequired",
      "component": "Authentication",
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
        // FIXME: Call token revoked list to check if token is revoked.
        log.WithFields(logrus.Fields{
          "fixme": 1,
        }).Warn("Missing check against token-revoked-list to check if token is revoked")

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
      "func": "main.authorizationRequired",
      "component": "Authorization",
    })

    log.Debug("Checking Authorization: Bearer <token> in request")

    _ /*accessToken*/, accessTokenExists := c.Get(environment.AccessTokenKey)
    if accessTokenExists == false {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "No access token found. Hint: Is bearer token missing?"})
      c.Abort()
      return
    }

    // FIXME: Implement QTNA #3 and #4

    // See #3 of QTNA
    log.WithFields(logrus.Fields{
      "fixme": 1,
    }).Warn("Missing check if access token is granted the required scopes")

    // See #4 of QTNA
    log.WithFields(logrus.Fields{
      "fixme": 1,
    }).Warn("Missing check if the user or client giving the grants in the access token authorized to use the scopes granted")

    strRequiredScopes := strings.Join(requiredScopes, ",")

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
