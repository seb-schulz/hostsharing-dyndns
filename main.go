package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"
	"reflect"

	"github.com/go-chi/chi/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type serveType int8

type serverConfig struct {
	ServerType     serveType
	HttpPort       string
	UpdaterHandler updaterHandlerConfig
}

const (
	serveTypeHttp serveType = iota
	serveTypeFcgi
)

func stringToServeTypeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(serveTypeHttp) {
			return data, nil
		}

		switch data.(string) {
		case "http":
			return serveTypeHttp, nil
		case "fcgi":
			return serveTypeFcgi, nil
		case "fastcgi":
			return serveTypeFcgi, nil
		default:
			return 0, fmt.Errorf("unknown server type")
		}
	}
}

func base64StringToBytesHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf([]byte{}) {
			return data, nil
		}

		if result, err := base64.URLEncoding.DecodeString(data.(string)); err == nil {
			return result, nil
		}

		return data, nil
	}
}

func loadServerConfig() (*serverConfig, error) {
	c := serverConfig{
		ServerType: serveTypeHttp,
		HttpPort:   "9000",
		UpdaterHandler: updaterHandlerConfig{
			Password: passwordConfig{
				KeyLen:  32,
				Threads: 4,
			},
		},
	}

	viper.SetConfigName(".hostsharing-dyndns.conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}

	if err := viper.Unmarshal(&c, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		base64StringToBytesHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		stringToServeTypeHookFunc(),
	))); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %v", err)
	}

	validationErrors := []error{}
	if c.UpdaterHandler.User == "" {
		validationErrors = append(validationErrors, fmt.Errorf("undefined user"))
	}

	if c.UpdaterHandler.Filename == "" {
		validationErrors = append(validationErrors, fmt.Errorf("undefined filename for zonefile"))
	}

	if c.UpdaterHandler.DomainSubpart == "" {
		validationErrors = append(validationErrors, fmt.Errorf("undefined domain subpart like HOME.dyndns.example.com"))
	}

	if len(c.UpdaterHandler.Password.Key) < 8 {
		validationErrors = append(validationErrors, fmt.Errorf("undefined/short password key"))
	}

	if len(c.UpdaterHandler.Password.Salt) < 8 {
		validationErrors = append(validationErrors, fmt.Errorf("undefined/short password salt"))
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("missing required configuration: \n\n%s", errors.Join(validationErrors...))
	}

	return &c, nil
}

var rootCmd = &cobra.Command{
	Use:   "hostsharing-dyndns",
	Short: "hostsharing-dyndns is a dyndns service for Hostsharing e.G.",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadServerConfig()
		if err != nil {
			return err
		}

		r := chi.NewRouter()
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello World")
		})
		r.Mount("/", updaterHandler(config.UpdaterHandler))

		switch config.ServerType {
		case serveTypeHttp:
			if config.HttpPort == "" {
				return fmt.Errorf("http port not defined")
			}

			port := config.HttpPort
			log.Println("Server listening on port ", port)
			if err := http.ListenAndServe(":"+port, r); err != nil {
				return fmt.Errorf("cannot run server: %v", err)
			}
		case serveTypeFcgi:
			if err := fcgi.Serve(nil, r); err != nil {
				return fmt.Errorf("cannot run server: %v", err)
			}
		default:
			panic("cannot run any server type")
		}
		return nil
	},
}

var validateConfigCmd = &cobra.Command{
	Use:   "validateConfig",
	Short: "validate configuration files",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadServerConfig()
		fmt.Println(c)
		return err
	},
}

func main() {
	rootCmd.AddCommand(validateConfigCmd, generatePasswordCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
