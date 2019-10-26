package main

import (
  "net/url"
  "os"
  "bufio"
  "io/ioutil"
  "golang.org/x/net/context"
  "golang.org/x/oauth2/clientcredentials"
  "github.com/sirupsen/logrus"
  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "github.com/pborman/getopt"
  "github.com/dgrijalva/jwt-go"
  "fmt"

  nats "github.com/nats-io/nats.go"

  "github.com/charmixer/idp/utils"
  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/migration"
  "github.com/charmixer/idp/endpoints/identities"
  "github.com/charmixer/idp/endpoints/humans"
  "github.com/charmixer/idp/endpoints/clients"
  "github.com/charmixer/idp/endpoints/challenges"
  "github.com/charmixer/idp/endpoints/invites"
  "github.com/charmixer/idp/endpoints/resourceservers"

  E "github.com/charmixer/idp/client/errors"
)

const app = "idp"

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

  E.InitRestErrors()
}

func createBanList(file string) (map[string]bool, error) {
  var banList map[string]bool = make(map[string]bool)
  f, err := os.Open(file)
  if err != nil {
    return nil, err
  }
  defer f.Close()

  scanner := bufio.NewScanner(f)
  for scanner.Scan() {
    banList[scanner.Text()] = true
  }

  if err := scanner.Err(); err != nil {
    return nil, err
  }

  return banList, nil
}

func migrate(driver neo4j.Driver) {
  migration.Migrate(driver)
}

func main() {

  optMigrate := getopt.BoolLong("migrate", 0, "Run migration")
  optServe := getopt.BoolLong("serve", 0, "Serve application")
  optHelp := getopt.BoolLong("help", 0, "Help")
  getopt.Parse()

  if *optHelp {
    getopt.Usage()
    os.Exit(0)
  }

  // https://medium.com/neo4j/neo4j-go-driver-is-out-fbb4ba5b3a30
  // Each driver instance is thread-safe and holds a pool of connections that can be re-used over time. If you don’t have a good reason to do otherwise, a typical application should have a single driver instance throughout its lifetime.
  log.WithFields(appFields).Debug("Fixme Neo4j loggning should go trough logrus so it does not differ in output from rest of the app")
  driver, err := neo4j.NewDriver(config.GetString("neo4j.uri"), neo4j.BasicAuth(config.GetString("neo4j.username"), config.GetString("neo4j.password"), ""), func(config *neo4j.Config) {
    config.Log = neo4j.ConsoleLogger(neo4j.DEBUG)

    /*if logDebug == 1 {
      config.Log = neo4j.ConsoleLogger(neo4j.DEBUG)
    } else {
      config.Log = neo4j.ConsoleLogger(neo4j.INFO)
    }*/
  });
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }
  defer driver.Close()

  // migrate then exit application
  if *optMigrate {
    migrate(driver)
    os.Exit(0)
    return
  }

  provider, err := oidc.NewProvider(context.Background(), config.GetString("hydra.public.url") + "/")
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  // Setup the hydra client idp is going to use (oauth2 client credentials)
  // NOTE: We store the hydraConfig also as we are going to need it to let idp app start the Oauth2 Authorization code flow.
  hydraConfig := &clientcredentials.Config{
    ClientID:     config.GetString("oauth2.client.id"),
    ClientSecret: config.GetString("oauth2.client.secret"),
    TokenURL:     provider.Endpoint().TokenURL,
    Scopes:       config.GetStringSlice("oauth2.scopes.required"),
    EndpointParams: url.Values{"audience": {"hydra"}},
    AuthStyle: 2, // https://godoc.org/golang.org/x/oauth2#AuthStyle
  }

  // Client to do protected client credentials calls to AAP like judge
  aapConfig := &clientcredentials.Config{
    ClientID:     config.GetString("oauth2.client.id"),
    ClientSecret: config.GetString("oauth2.client.secret"),
    TokenURL:     provider.Endpoint().TokenURL,
    Scopes:       config.GetStringSlice("oauth2.scopes.required"),
    EndpointParams: url.Values{"audience": {"aap"}},
    AuthStyle: 2, // https://godoc.org/golang.org/x/oauth2#AuthStyle
  }


  bannedUsernames, err := createBanList("/ban/usernames")
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  // Load private and public key for signing jwt tokens.
  signBytes, err := ioutil.ReadFile(config.GetString("serve.tls.key.path"))
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  verifyBytes, err := ioutil.ReadFile(config.GetString("serve.tls.cert.path"))
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }

  natsConnection, err := nats.Connect(config.GetString("nats.url"))
  if err != nil {
    log.WithFields(appFields).Panic(err.Error())
    return
  }
  defer natsConnection.Close()

  // Setup app state variables. Can be used in handler functions by doing closures see exchangeAuthorizationCodeCallback
  env := &environment.State{
    Provider: provider,
    HydraConfig: hydraConfig,
    AapConfig: aapConfig,
    Driver: driver,
    BannedUsernames: bannedUsernames,
    IssuerSignKey: signKey,
    IssuerVerifyKey: verifyKey,
    Nats: natsConnection,
  }

  if *optServe {
    serve(env)
  } else {
    getopt.Usage()
    os.Exit(0)
  }

}

func requestBeforeAuth() gin.HandlerFunc {
  return func(c *gin.Context) {
		fmt.Println(c.Request)
		c.Next()
	}
}

func requestAfterAuth() gin.HandlerFunc {
  return func(c *gin.Context) {
		fmt.Println(c.Request)
		c.Next()
	}
}

func serve(env *environment.State) {

  r := gin.New() // Clean gin to take control with logging.
  r.Use(gin.Recovery())
  r.Use(utils.ProcessMethodOverride(r))
  r.Use(utils.RequestId())
  r.Use(utils.RequestLogger(environment.LogKey, environment.RequestIdKey, log, appFields))

  // ## QTNA - Questions that need answering before granting access to a protected resource
  // 1. Is the user or client authenticated? Answered by the process of obtaining an access token.
  // 2. Is the access token expired?
  // 3. Is the access token granted the required scopes?
  // 4. Is the user or client giving the grants in the access token authorized to operate the scopes granted?
  // 5. Is the access token revoked?

  // All requests need to be authenticated.
  r.Use(utils.AuthenticationRequired(environment.LogKey, environment.AccessTokenKey))

  hydraIntrospectUrl := config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.introspect")

  aconf := utils.AuthorizationConfig{
    LogKey:             environment.LogKey,
    AccessTokenKey:     environment.AccessTokenKey,
    HydraConfig:        env.HydraConfig,
    HydraIntrospectUrl: hydraIntrospectUrl,
    AapConfig:          env.AapConfig,
  }

  r.GET(  "/challenges",       utils.AuthorizationRequired(aconf, "idp:read:challenges"),         challenges.GetChallenges(env) )
  r.POST( "/challenges",       utils.AuthorizationRequired(aconf, "idp:create:challenges"),        challenges.PostChallenges(env) )
  r.PUT( "/challenges/verify", utils.AuthorizationRequired(aconf, "idp:update:challenges:verify"), challenges.PutVerify(env) )

  r.GET(    "/identities",     utils.AuthorizationRequired(aconf, "idp:read:identities"), identities.GetIdentities(env) )

  r.GET(    "/humans", utils.AuthorizationRequired(aconf, "idp:read:humans"), humans.GetHumans(env))
  r.POST(   "/humans", utils.AuthorizationRequired(aconf, "idp:create:humans"), humans.PostHumans(env) )
  r.PUT(    "/humans", utils.AuthorizationRequired(aconf, "idp:update:humans"), humans.PutHumans(env) )
  r.DELETE( "/humans", utils.AuthorizationRequired(aconf, "idp:delete:humans"), humans.DeleteHumans(env) )

  r.POST( "/humans/authenticate", utils.AuthorizationRequired(aconf, "idp:create:humans:authenticate"), humans.PostAuthenticate(env) )
  r.PUT(  "/humans/password", utils.AuthorizationRequired(aconf, "idp:update:humans:password"), humans.PutPassword(env) )

  r.PUT(  "/humans/totp", utils.AuthorizationRequired(aconf, "idp:update:humans:totp"), humans.PutTotp(env) )

  r.GET(  "/humans/logout", utils.AuthorizationRequired(aconf, "idp:read:humans:logout"),    humans.GetLogout(env) )
  r.POST( "/humans/logout", utils.AuthorizationRequired(aconf, "idp:create:humans:logout"),  humans.PostLogout(env) )
  r.PUT(  "/humans/logout",  utils.AuthorizationRequired(aconf, "idp:update:humans:logout"), humans.PutLogout(env) )

  r.PUT(  "/humans/deleteverification", utils.AuthorizationRequired(aconf, "idp:update:humans:deleteverification"), humans.PutDeleteVerification(env) )

  r.POST( "/humans/recover", utils.AuthorizationRequired(aconf, "idp:create:humans:recover"), humans.PostRecover(env) )
  r.PUT(  "/humans/recoververification", utils.AuthorizationRequired(aconf, "idp:update:humans:recoververification"), humans.PutRecoverVerification(env) )

  r.GET ( "/clients", utils.AuthorizationRequired(aconf, "idp:read:clients"), clients.GetClients(env))
  r.POST( "/clients", utils.AuthorizationRequired(aconf, "idp:create:clients"), clients.PostClients(env) )
  r.DELETE( "/clients", utils.AuthorizationRequired(aconf, "idp:delete:clients"), clients.DeleteClients(env) )

  r.GET ( "/resourceservers", utils.AuthorizationRequired(aconf, "idp:read:resourceservers"), resourceservers.GetResourceServers(env))
  r.POST( "/resourceservers", utils.AuthorizationRequired(aconf, "idp:create:resourceservers"), resourceservers.PostResourceServers(env) )
  r.DELETE( "/resourceservers", utils.AuthorizationRequired(aconf, "idp:delete:resourceservers"), resourceservers.DeleteResourceServers(env) )

  r.GET(  "/invites", utils.AuthorizationRequired(aconf, "idp:read:invites"), invites.GetInvites(env) )
  r.POST( "/invites", utils.AuthorizationRequired(aconf, "idp:create:invites"), invites.PostInvites(env) )
  r.POST( "/invites/send", utils.AuthorizationRequired(aconf, "idp:send:invites"), invites.PostInvitesSend(env) )
  r.POST( "/invites/claim", utils.AuthorizationRequired(aconf, "idp:claim:invites"), invites.PostInvitesClaim(env) )

  r.RunTLS(":" + config.GetString("serve.public.port"), config.GetString("serve.tls.cert.path"), config.GetString("serve.tls.key.path"))
}