package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mitchellh/mapstructure"
	"github.com/sebatec-eu/config-mate/hostsharing"
	"github.com/spf13/cobra"
)

type serverConfig struct {
	UpdaterHandler updaterHandlerConfig
	Logger         struct {
		Enabled bool
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
		UpdaterHandler: updaterHandlerConfig{
			Password: passwordConfig{
				KeyLen:  32,
				Threads: 4,
			},
		},
	}

	if err := hostsharing.FcgiReadInConfig(&c, base64StringToBytesHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	); err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
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
		if config.Logger.Enabled {
			r.Use(hostsharing.RequestLogger("hostsharing-dyndns"))
		}
		r.Use(middleware.Heartbeat("/ping"))
		r.Mount("/", updaterHandler(config.UpdaterHandler))

		if err := hostsharing.ListenAndServe(r); err != nil {
			return err
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
