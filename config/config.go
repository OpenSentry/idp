package config

import (
  "github.com/spf13/viper"
  "fmt"
  "strings"
)

type DiscoveryConfig struct {
  IdpUi struct {
    Public struct {
      Url  string
      Port string
      Endpoints struct {
      }
    }
  }
  IdpApi struct {
    Public struct {
      Url  string
      Port string
      Endpoints struct {
        Authenticate string
        Identities string
        Logout string
      }
    }
  }
  AapUi struct {
    Public struct {
      Url  string
      Port string
      Endpoints struct {
      }
    }
  }
  AapApi struct {
    Public struct {
      Url  string
      Port string
      Endpoints struct {
        Authorizations string
        AuthorizationsAuthorize string
        AuthorizationsReject string
      }
    }
  }
  Hydra struct {
    Public struct {
      Url  string
      Port string
      Endpoints struct {
        Oauth2Token string
        Oauth2Auth string
        Userinfo string
        HealthAlive string
        HealthReady string
        Logout string
      }
    }
    Private struct {
      Url  string
      Port string
      Endpoints struct {
        Consent string
        ConsentAccept string
        ConsentReject string
        Login string
        LoginAccept string
        LoginReject string
        Logout string
        LogoutAccept string
        LogoutReject string
      }
    }
  }
}

type AppConfig struct {
  Serve struct {
    Public struct {
      Port string
    }
    Tls struct {
      Key struct {
        Path string
      }
      Cert struct {
        Path string
      }
    }
  }
  Neo4j struct {
    Uri string
    Username string
    Password string
  }
  Csrf struct {
    AuthKey string
  }
  Oauth2 struct {
    Client struct {
      Id string
      Secret string
    }
    Scopes struct {
      Required []string
    }
  }
}

var Discovery DiscoveryConfig
var App AppConfig

func setDefaults() {
  viper.SetDefault("config.discovery.path", "./discovery.yml")
  viper.SetDefault("config.app.path", "./app.yml")
}

func InitConfigurations() {
  var err error

  // lets environment variable override config file
  viper.AutomaticEnv()
  viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

  setDefaults()

  // Load discovery configurations

  viper.SetConfigFile(viper.GetString("config.discovery.path"))
  err = viper.ReadInConfig() // Find and read the config file
  if err != nil { // Handle errors reading the config file
    panic(fmt.Errorf("Fatal error config file: %s \n", err))
  }

  err = viper.Unmarshal(&Discovery)
  if err != nil {
    fmt.Printf("unable to decode into config struct, %v", err)
  }

  // Load app specific configurations

  viper.SetConfigFile(viper.GetString("config.app.path"))
  err = viper.ReadInConfig() // Find and read the config file
  if err != nil { // Handle errors reading the config file
    panic(fmt.Errorf("Fatal error config file: %s \n", err))
  }

  err = viper.Unmarshal(&App)
  if err != nil {
    fmt.Printf("unable to decode into config struct, %v", err)
  }
}
