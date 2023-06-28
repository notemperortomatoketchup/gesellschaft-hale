package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"google.golang.org/grpc"
)

type Application struct {
	Clients   sync.Map
	RequestCh chan *protocol.RequestJobWrapper
	Results   sync.Map
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
	app := &Application{
		RequestCh: make(chan *protocol.RequestJobWrapper, 5),
	}

	l, err := net.Listen("tcp", ":50001")
	if err != nil {
		log.Fatalf("err listener: %v", err)
	}

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

	go app.initServer()
	go app.initAPI()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM, syscall.SIGINT)
	<-terminate
}
