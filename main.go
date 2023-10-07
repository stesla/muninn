package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stesla/telnet"
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
	api.GET("/connect/:address", connect)
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type message struct {
	messageType int
	data        []byte
}

func connect(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	address := params.ByName("address")
	upstream, err := telnet.Dial(address)
	if err != nil {
		log.Warn().Str("address", address).Err(err).Msg("error connecting to address")
		http.Error(w, "error connecting to address", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn().Err(err).Msg("upgrade")
		return
	}
	defer c.Close()

	closech := make(chan struct{})
	defer close(closech)

	errch := make(chan error, 1)

	go func(r io.Reader, downstream *websocket.Conn) {
		scanner := bufio.NewScanner(r)
		for {
			select {
			case <-closech:
				return
			default:
			}
			if scanner.Scan() {
				downstream.WriteMessage(websocket.TextMessage, scanner.Bytes())
				log.Debug().Str("bytes", string(scanner.Bytes())).Msg("sent")
			} else if err := scanner.Err(); err != nil {
				log.Warn().Err(err).Msg("read upstream")
				errch <- err
				return
			}
		}
	}(upstream, c)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Warn().Err(err).Msg("read downstream")
			return
		}
		log.Debug().
			Int("type", mt).
			Str("bytes", string(message)).
			Msg("recieved message")
		switch mt {
		case websocket.TextMessage:
			_, err = upstream.Write(append(message, '\n'))
			if err != nil {
				log.Warn().Err(err).Msg("write upstream")
				return
			}
		default:
		}
	}
}
