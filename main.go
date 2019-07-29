package main

import (
  "strings"
  "net/http"
  "net/url"


  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"

  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"
  "github.com/atarantini/ginrequestid"

  "github.com/neo4j/neo4j-go-driver/neo4j"

  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/identities"
)

const app = "idpbe"

func init() {
  config.InitConfigurations()
}

func main() {
  // https://medium.com/neo4j/neo4j-go-driver-is-out-fbb4ba5b3a30
  // Each driver instance is thread-safe and holds a pool of connections that can be re-used over time. If you don’t have a good reason to do otherwise, a typical application should have a single driver instance throughout its lifetime.
  driver, err := neo4j.NewDriver(config.App.Neo4j.Uri, neo4j.BasicAuth(config.App.Neo4j.Username, config.App.Neo4j.Password, ""), func(config *neo4j.Config) {
    config.Log = neo4j.ConsoleLogger(neo4j.DEBUG)
  });
  if err != nil {
    environment.DebugLog(app, "main", "[database:Neo4j] " + err.Error(), "")
    return
  }
  defer driver.Close()

  provider, err := oidc.NewProvider(context.Background(), config.Discovery.Hydra.Public.Url + "/")
  if err != nil {
    environment.DebugLog(app, "main", "[provider:hydra] " + err.Error(), "")
    return
  }

  // Setup the hydra client idpbe is going to use (oauth2 client credentials)
  // NOTE: We store the hydraConfig also as we are going to need it to let idpbe app start the Oauth2 Authorization code flow.
  hydraConfig := &clientcredentials.Config{
    ClientID:     config.App.Oauth2.Client.Id,
    ClientSecret: config.App.Oauth2.Client.Secret,
    TokenURL:     provider.Endpoint().TokenURL,
    Scopes:       config.App.Oauth2.Scopes.Required,
    EndpointParams: url.Values{"audience": {"hydra"}},
    AuthStyle: 2, // https://godoc.org/golang.org/x/oauth2#AuthStyle
  }

  // Setup app state variables. Can be used in handler functions by doing closures see exchangeAuthorizationCodeCallback
  env := &environment.State{
    Provider: provider,
    HydraConfig: hydraConfig,
    Driver: driver,
  }

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

  r.RunTLS(":" + config.App.Serve.Public.Port, config.App.Serve.Tls.Cert.Path, config.App.Serve.Tls.Key.Path)
}

func authenticationRequired() gin.HandlerFunc {
  fn := func(c *gin.Context) {
    var requestId string = c.MustGet(environment.RequestIdKey).(string)
    environment.DebugLog(app, "authenticationRequired", "Checking Authorization: Bearer <token> in request", requestId)

    var token *oauth2.Token
    auth := c.Request.Header.Get("Authorization")
    split := strings.SplitN(auth, " ", 2)
    if len(split) == 2 || strings.EqualFold(split[0], "bearer") {
      environment.DebugLog(app, "authenticationRequired", "Authorization: Bearer <token> found for request.", requestId)
      token = &oauth2.Token{
        AccessToken: split[1],
        TokenType: split[0],
      }

      // See #2 of QTNA
      // https://godoc.org/golang.org/x/oauth2#Token.Valid
      if token.Valid() == true {

        // See #5 of QTNA
        // FIXME: Call token revoked list to check if token is revoked.

        environment.DebugLog(app, "authenticationRequired", "Valid access token", requestId)
        c.Set(environment.AccessTokenKey, token)
        c.Next() // Authentication successful, continue.
        return;
      }

      // Deny by default
      environment.DebugLog(app, "authenticationRequired", "Invalid Access token", requestId)
      c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token."})
      c.Abort()
      return
    }

    // Deny by default
    environment.DebugLog(app, "authenticationRequired", "Missing access token", requestId)
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization: Bearer <token> not found in request."})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func authorizationRequired(route environment.Route, requiredScopes ...string) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    var requestId string = c.MustGet(environment.RequestIdKey).(string)
    environment.DebugLog(app, "authorizationRequired", "Checking Authorization: Bearer <token> in request", requestId)

    _ /*accessToken*/, accessTokenExists := c.Get(environment.AccessTokenKey)
    if accessTokenExists == false {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "No access token found. Hint: Is bearer token missing?"})
      c.Abort()
      return
    }

    // FIXME: Implement QTNA #3 and #4

    // See #3 of QTNA
    environment.DebugLog(app, "authorizationRequired", "Missing implementation of QTNA #3 - Is the access token granted the required scopes?", requestId)
    // See #4 of QTNA
    environment.DebugLog(app, "authorizationRequired", "Missing implementation of QTNA #4 - Is the user or client giving the grants in the access token authorized to operate the scopes granted?", requestId)

    strRequiredScopes := strings.Join(requiredScopes, ",")

    foundRequiredScopes := true
    if foundRequiredScopes {
      environment.DebugLog(app, "authorizationRequired", "Valid scopes: " + strRequiredScopes, requestId)
      c.Next() // Authentication successful, continue.
      return;
    }

    // Deny by default
    environment.DebugLog(app, "authorizationRequired", "Invalid scopes: " + strRequiredScopes + " Hint: Some required scopes are missing, invalid or not granted", requestId)
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid scopes. Hint: Some required scopes are missing, invalid or not granted"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}
