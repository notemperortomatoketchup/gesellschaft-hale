package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
	"google.golang.org/grpc"
)

type Application struct {
	UseJWT    bool
	Clients   sync.Map
	RequestCh chan *protocol.RequestJobWrapper
	Results   sync.Map
	Fiber     *fiber.App
}

type Server struct {
	protocol.UnimplementedHalerServer
	app *Application
}

type Client struct {
	id    uint32
	slots int32
}

var jwtsecret = []byte("HelloJWT3030033")

func main() {
	app := &Application{
		RequestCh: make(chan *protocol.RequestJobWrapper, 5),
	}

	flag.BoolVar(&app.UseJWT, "jwt", false, "ues jwt or not?")
	flag.Parse()

	l, err := net.Listen("tcp", ":50001")
	if err != nil {
		log.Fatalf("err listener: %v", err)
	}

	models.StartDB()
	app.StartValidator()

	s := grpc.NewServer()
	protocol.RegisterHalerServer(s, &Server{
		app: app,
	})

	log.Printf("server listening at %v", l.Addr())

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

	if err := app.Fiber.ShutdownWithTimeout(45 * time.Second); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}
