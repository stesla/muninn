package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

//go:embed all:nextjs/dist
var nextFS embed.FS

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	pflag.String("address", ":3001", "address to which we should bind")
	viper.BindPFlag("http.address", pflag.Lookup("address"))

	config := pflag.String("config", "", "path to config file")

	pflag.Parse()

	viper.SetEnvPrefix("muninn")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if *config != "" {
		viper.SetConfigFile(*config)
	}
	viper.SetConfigName("muninn")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.muninn")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	switch err.(type) {
	case viper.ConfigFileNotFoundError:
	default:
		panic(err)
	}

	distFS, err := fs.Sub(nextFS, "nextjs/dist")
	if err != nil {
		panic(err)
	}

	api := httprouter.New()
	api.GET("/ping", httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		pong := map[string]string{"ping": "pong"}
		json.NewEncoder(w).Encode(&pong)
	}))

	http.Handle("/", http.FileServer(http.FS(distFS)))
	http.Handle("/api/", http.StripPrefix("/api", api))

	log.Info().
		Str("http.address", viper.GetString("http.address")).
		Msg("started")

	err = http.ListenAndServe(viper.GetString("http.address"), nil)
	log.Fatal().Err(err).Msg("")
}
