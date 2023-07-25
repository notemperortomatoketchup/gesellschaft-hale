package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
	"google.golang.org/grpc"
)

type Application struct {
	clients   sync.Map
	requestCh chan *protocol.RequestJobWrapper
	results   sync.Map
	fiber     *fiber.App
	config    *Config
}

type Config struct {
	CoreConfig
	JWTConfig
	DatabaseConfig
}

type CoreConfig struct {
	port string
	dev  bool
}

type DatabaseConfig struct {
	dsn string
}

type JWTConfig struct {
	enabled bool
	secret  []byte
}

type Server struct {
	protocol.UnimplementedHalerServer
	app *Application
}

type Client struct {
	id    uint32
	slots int32
}

func main() {
	l, err := net.Listen("tcp", ":50001")
	if err != nil {
		log.Fatalf("err listener: %v", err)
	}

	app := &Application{
		requestCh: make(chan *protocol.RequestJobWrapper, 5),
		config:    StartConfig(),
	}
	models.StartDB(app.config.dsn)

	s := grpc.NewServer()
	protocol.RegisterHalerServer(s, &Server{
		app: app,
	})

	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("err serving: %v", err)
		}
	}()

	go app.StartServer()
	go app.StartAPI()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM, syscall.SIGINT)
	<-terminate

	// if err := app.fiber.ShutdownWithTimeout(45 * time.Second); err != nil {
	// 	log.Fatalf("graceful shutdown failed: %v", err)
	// }
}

func StartConfig() *Config {
	config := new(Config)

	viper.AddConfigPath("../../../configurations/")
	viper.AddConfigPath(".")
	viper.SetConfigName("server")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("err reading config: %v", err)
	}

	config.port = viper.GetString("core.port")
	config.dev = viper.GetBool("core.dev")

	config.dsn = viper.GetString("database.dsn")

	config.enabled = viper.GetBool("jwt.enabled")
	config.secret = []byte(viper.GetString("jwt.secret"))

	return config
}
