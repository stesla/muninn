package main

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templateFS embed.FS

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	pflag.String("address", ":8080", "address to which we should bind")
	viper.BindPFlag("http.address", pflag.Lookup("address"))
	viper.SetDefault("http.address", ":8080")

	viper.SetDefault("embedded-files", true)

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
	if err != nil {
		panic(err)
	}

	r := gin.New()
	r.SetTrustedProxies(nil)

	rootTemplate := template.New("root").Funcs(r.FuncMap)
	r.SetHTMLTemplate(template.Must(rootTemplate.ParseFS(templateFS, "templates/*")))
	r.Use(
		gin.Recovery(),
		Logger(),
	)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	staticFiles, _ := fs.Sub(staticFS, "static")
	r.StaticFS("/static", http.FS(staticFiles))

	addr := viper.GetString("http.address")
	log.Info().
		Str("http.address", addr).
		Msg("starting gin server")
	r.Run(viper.GetString("http.address"))
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		c.Next()

		var event *zerolog.Event
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			event = log.Error()
		case statusCode >= 400 && statusCode < 500:
			event = log.Warn()
		default:
			event = log.Info()
		}
		event.
			Str("client_ip", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("http_status", statusCode).
			Strs("errors", c.Errors.Errors()).
			Msg("")
	}
}
