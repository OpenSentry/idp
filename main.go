package main

import (
  "net/url"
  "os"
  "bufio"
  "runtime"
  "path"
  "io/ioutil"
  "golang.org/x/net/context"
  "golang.org/x/oauth2/clientcredentials"
  "github.com/sirupsen/logrus"
  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
	"database/sql"

  "github.com/pborman/getopt"
  "github.com/dgrijalva/jwt-go"
  "fmt"

  nats "github.com/nats-io/nats.go"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/config"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/migration"
  "github.com/opensentry/idp/endpoints/identities"
  "github.com/opensentry/idp/endpoints/humans"
  "github.com/opensentry/idp/endpoints/clients"
  "github.com/opensentry/idp/endpoints/challenges"
  "github.com/opensentry/idp/endpoints/invites"
  "github.com/opensentry/idp/endpoints/resourceservers"
  "github.com/opensentry/idp/endpoints/roles"

  E "github.com/opensentry/idp/client/errors"
)

const appName = "idp"

var (
  logDebug int // Set to 1 to enable debug
  logFormat string // Current only supports default and json

  log *logrus.Logger

  appFields logrus.Fields

  templateMap map[idp.ChallengeType]app.EmailTemplate = make(map[idp.ChallengeType]app.EmailTemplate)
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

  log.SetReportCaller(true)
  log.Formatter = &logrus.TextFormatter{
    CallerPrettyfier: func(f *runtime.Frame) (string, string) {
      filename := path.Base(f.File)
      return "", fmt.Sprintf("%s:%d", filename, f.Line)
    },
  }

  // We only have 2 log levels. Things developers care about (debug) and things the user of the app cares about (info)
  if logDebug == 1 {
    log.SetLevel(logrus.DebugLevel)
  } else {
    log.SetLevel(logrus.InfoLevel)
  }
  if logFormat == "json" {
    log.SetFormatter(&logrus.JSONFormatter{})
  }

  appFields = logrus.Fields{
    "appname": appName,
    "log.debug": logDebug,
    "log.format": logFormat,
  }

  setupTemplateMap()

  E.InitRestErrors()
}

func setupTemplateMap() {
  senderName := config.GetString("provider.name")
  if senderName == "" {
    log.Panic("Missing config provider.name")
    return
  }

  senderEmail := config.GetString("provider.email")
  if (senderEmail == "") {
    log.Panic("Missing config provider.email")
    return
  }
  sender := idp.SMTPSender{ Name:senderName, Email:senderEmail }

  baseKey := "templates"

  challenges := map[idp.ChallengeType]string{
    idp.ChallengeAuthenticate: "authenticate",
    idp.ChallengeRecover: "recover",
    idp.ChallengeDelete: "delete",
    idp.ChallengeEmailConfirm: "emailconfirm",
    idp.ChallengeEmailChange: "emailchange",
  }

  for ct, challengeKey := range challenges {

    key := baseKey + "." + challengeKey + ".email.templatefile"
    var templateFile string = config.GetString(key)
    if templateFile == "" {
      log.Panic("Missing config " + key)
      return
    }

    key = baseKey + "." + challengeKey + ".email.subject"
    var subject string = config.GetString(key)
    if subject == "" {
      log.Panic("Missing config " + key)
      return
    }

    templateMap[ct] = app.EmailTemplate{
      Sender: sender,
      File: templateFile,
      Subject: subject,
    }
  }
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

func migrate(driver *sql.DB) {
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

	// Create DB pool
	driver, err := sql.Open("postgres", config.GetString("db.dsn"))
	if err != nil {
		fmt.Println(err)
		panic("Failed to open a DB connection")
	}
	/*if debug == 1 {
		// TODO postgres logging settings
	}*/
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
  env := &app.Environment{
    Constants: &app.EnvironmentConstants{
      RequestIdKey: "RequestId",
      LogKey: "log",
      AccessTokenKey: "access_token",
      IdTokenKey: "id_token",

      ContextAccessTokenKey: "access_token",
      ContextIdTokenKey: "id_token",
      ContextIdTokenHintKey: "id_token_hint",
      ContextIdentityKey: "id",
      ContextOAuth2ConfigKey: "oauth2_config",
      ContextRequiredScopesKey: "required_scopes",
      ContextPrecalculatedStateKey: "precalculated_state",
    },

    Provider: provider,
    HydraConfig: hydraConfig,
    AapConfig: aapConfig,
    Driver: driver,
    BannedUsernames: bannedUsernames,
    IssuerSignKey: signKey,
    IssuerVerifyKey: verifyKey,
    Nats: natsConnection,
    TemplateMap: &templateMap,
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

func serve(env *app.Environment) {

  r := gin.New() // Clean gin to take control with logging.
  r.Use(gin.Recovery())
  r.Use(app.ProcessMethodOverride(r))
  r.Use(app.RequestId())
  r.Use(app.RequestLogger(env.Constants.LogKey, env.Constants.RequestIdKey, log, appFields))

  // ## QTNA - Questions that need answering before granting access to a protected resource
  // 1. Is the user or client authenticated? Answered by the process of obtaining an access token.
  // 2. Is the access token expired?
  // 3. Is the access token granted the required scopes?
  // 4. Is the user or client giving the grants in the access token authorized to operate the scopes granted?
  // 5. Is the access token revoked?

  // All requests need to be authenticated.
  r.Use(app.AuthenticationRequired(env.Constants.LogKey, env.Constants.AccessTokenKey))

  hydraIntrospectUrl := config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.introspect")

  aconf := app.AuthorizationConfig{
    LogKey:             env.Constants.LogKey,
    AccessTokenKey:     env.Constants.AccessTokenKey,
    HydraConfig:        env.HydraConfig,
    HydraIntrospectUrl: hydraIntrospectUrl,
    AapConfig:          env.AapConfig,
  }

  // TODO: Maybe instaed of letting the enpoint do scope requirements on confirmation_type, that should be part of the set up here aswell, but intertwined with the input data somehow?
  r.GET(  "/challenges",       app.AuthorizationRequired(aconf, "idp:read:challenges"),         challenges.GetChallenges(env) )
  r.POST( "/challenges",       app.AuthorizationRequired(aconf, "idp:create:challenges"),        challenges.PostChallenges(env) )
  r.PUT( "/challenges/verify", app.AuthorizationRequired(aconf, "idp:update:challenges:verify"), challenges.PutVerify(env) )

  r.GET(    "/identities",     app.AuthorizationRequired(aconf, "idp:read:identities"), identities.GetIdentities(env) )

  r.GET(    "/humans", app.AuthorizationRequired(aconf, "idp:read:humans"), humans.GetHumans(env))
  r.POST(   "/humans", app.AuthorizationRequired(aconf, "idp:create:humans"), humans.PostHumans(env) )
  r.PUT(    "/humans", app.AuthorizationRequired(aconf, "idp:update:humans"), humans.PutHumans(env) )

  r.DELETE( "/humans", app.AuthorizationRequired(aconf, "idp:delete:humans"), humans.DeleteHumans(env) )

  r.POST( "/humans/authenticate", app.AuthorizationRequired(aconf, "idp:create:humans:authenticate"), humans.PostAuthenticate(env) )
  r.PUT(  "/humans/password", app.AuthorizationRequired(aconf, "idp:update:humans:password"), humans.PutPassword(env) )

  r.PUT(  "/humans/totp", app.AuthorizationRequired(aconf, "idp:update:humans:totp"), humans.PutTotp(env) )
  r.PUT(  "/humans/email", app.AuthorizationRequired(aconf, "idp:update:humans:email"), humans.PutEmail(env) )

  r.GET(  "/humans/logout", app.AuthorizationRequired(aconf, "idp:read:humans:logout"),    humans.GetLogout(env) )
  r.POST( "/humans/logout", app.AuthorizationRequired(aconf, "idp:create:humans:logout"),  humans.PostLogout(env) )
  r.PUT(  "/humans/logout",  app.AuthorizationRequired(aconf, "idp:update:humans:logout"), humans.PutLogout(env) )

  r.PUT(  "/humans/deleteverification", app.AuthorizationRequired(aconf, "idp:update:humans:deleteverification"), humans.PutDeleteVerification(env) )

  r.POST( "/humans/recover", app.AuthorizationRequired(aconf, "idp:create:humans:recover"), humans.PostRecover(env) )
  r.PUT(  "/humans/recoververification", app.AuthorizationRequired(aconf, "idp:update:humans:recoververification"), humans.PutRecoverVerification(env) )

  r.POST( "/humans/emailchange", app.AuthorizationRequired(aconf, "idp:create:humans:emailchange"), humans.PostEmailChange(env) )
  r.PUT(  "/humans/emailchange", app.AuthorizationRequired(aconf, "idp:update:humans:emailchange"), humans.PutEmailChange(env) )

  r.GET ( "/clients", app.AuthorizationRequired(aconf, "idp:read:clients"), clients.GetClients(env))
  r.POST( "/clients", app.AuthorizationRequired(aconf, "idp:create:clients"), clients.PostClients(env) )
  r.DELETE( "/clients", app.AuthorizationRequired(aconf, "idp:delete:clients"), clients.DeleteClients(env) )

  r.GET ( "/resourceservers", app.AuthorizationRequired(aconf, "idp:read:resourceservers"), resourceservers.GetResourceServers(env))
  r.POST( "/resourceservers", app.AuthorizationRequired(aconf, "idp:create:resourceservers"), resourceservers.PostResourceServers(env) )
  r.DELETE( "/resourceservers", app.AuthorizationRequired(aconf, "idp:delete:resourceservers"), resourceservers.DeleteResourceServers(env) )

  r.GET ( "/roles", app.AuthorizationRequired(aconf, "idp:read:roles"), roles.GetRoles(env))
  r.POST( "/roles", app.AuthorizationRequired(aconf, "idp:create:roles"), roles.PostRoles(env) )
  r.DELETE( "/roles", app.AuthorizationRequired(aconf, "idp:delete:roles"), roles.DeleteRoles(env) )

  r.GET(  "/invites", app.AuthorizationRequired(aconf, "idp:read:invites"), invites.GetInvites(env) )
  r.POST( "/invites", app.AuthorizationRequired(aconf, "idp:create:invites"), invites.PostInvites(env) )
  r.POST( "/invites/send", app.AuthorizationRequired(aconf, "idp:create:invites:send"), invites.PostInvitesSend(env) )
  r.POST( "/invites/claim", app.AuthorizationRequired(aconf, "idp:create:invites:claim"), invites.PostInvitesClaim(env) )

  r.RunTLS(":" + config.GetString("serve.public.port"), config.GetString("serve.tls.cert.path"), config.GetString("serve.tls.key.path"))
}
