package cmd

import (
	"log"
	"os"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"

	"github.com/jessevdk/go-flags"
)

type Cli struct {
	ServeCmd `command:"serve" description:"serves the application"`
}

type App struct {
	Cli

	Config struct {
		Ban struct {
			Usernames string
		}

		Crypto struct {
			Keys struct {
				Clients []string
				Totp    []string
			}
		}

		Csrf struct {
			AuthKey string
		}

		Hydra struct {
			Session struct {
				Timeout int
			}
		}

		Id string

		Invite struct {
			Template struct {
				Email struct {
					File    string
					Subject string
				}
			}
			Ttl int
			Url string
		}

		Log struct {
			Debug  bool
			Format string
		}

		Mail struct {
			Smtp struct {
				Host          string
				Password      string
				SkipTlsVerify bool
				User          string
			}
		}

		Migration struct {
			Data struct {
				Path string
			}
			Schema struct {
				Path string
			}
		}

		Nats struct {
			Url string
		}

		Neo4j struct {
			Debug    bool
			Password string
			Uri      string
			Username string
		}

		Oauth2 struct {
			Client struct {
				Id     string
				Secret string
			}
			Scopes struct {
				Required []string
			}
		}

		Provider struct {
			Email string
			Name  string
		}

		Serve struct {
			Public struct {
				Port int
			}
			TLS struct {
				Cert struct {
					Path string
				}
				Key struct {
					Path string
				}
			}
		}

		Session struct {
			Authkey string
		}

		Templates struct {
			Authenticate struct {
				Email struct {
					Subject      string
					Templatefile string
				}
			}
			Delete struct {
				Email struct {
					Subject      string
					Templatefile string
				}
			}
			Emailchange struct {
				Email struct {
					Subject      string
					Templatefile string
				}
			}
			Emailconfirm struct {
				Email struct {
					Subject      string
					Templatefile string
				}
			}
			Recover struct {
				Email struct {
					Subject      string
					Templatefile string
				}
			}
		}

		Totp struct {
			Cryptkey string
		}
	}
}

var Application App

func init() {

	configFile := os.Getenv("CFG_PATH")

	files := []string{}
	if configFile != "" {
		files = append(files, configFile)
	}

	loader := aconfig.LoaderFor(&Application.Config, aconfig.Config{
		SkipEnv:   false,
		SkipFlags: true,
		Files:     files,
		FileDecoders: map[string]aconfig.FileDecoder{
			".yml": aconfigyaml.New(),
		},
		AllowUnknownFields: true, // Maybe set to false for strict config files
		AllowUnknownEnvs:   true,
		EnvPrefix:          "CFG",
		FailOnFileNotFound: true,
	})

	if err := loader.Load(); err != nil {
		log.Panic(err)
	}

	flags.Parse(&Application)

	os.Exit(1)
}

/*
func main() {

	var cli Cli
	var cfg Config
	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		SkipEnv:   false,
		SkipFlags: true,
		Files:     []string{"config.yaml"},
		FileDecoders: map[string]aconfig.FileDecoder{
			".yaml": aconfigyaml.New(),
		},
		AllowUnknownFields: true, // Maybe set to false for strict config files
		EnvPrefix:    "CFG",
	})

	if err := loader.Load(); err != nil {
		log.Panic(err)
	}

	// Echo command
	gocmd.HandleFlag("Echo", func(cmd *gocmd.Cmd, args []string) error {
		fmt.Printf("%v\n", cmd.FlagValue("Echo.Out"))

		fmt.Printf("Db.Host: %v\n", cfg.Database.Host)
		fmt.Printf("Db.User: %v\n", cfg.Database.User)
		fmt.Printf("Db.Pass: %v\n", cfg.Database.Pass)

		fmt.Printf("Crypto.Keys.Totp: %v\n", cfg.Crypto.Keys.Totp)
		fmt.Printf("Crypto.Keys.Clients: %v\n", cfg.Crypto.Keys.Clients)

		return nil
	})
	// Init the app
	gocmd.New(gocmd.Options{
		Name:        "basic",
		Description: "A basic app",
		Version:     fmt.Sprintf("%s (%s)", "0.0.0", "..."),
		Flags:       &cli,
		ConfigType:  gocmd.ConfigTypeAuto,
	})

}
*/
