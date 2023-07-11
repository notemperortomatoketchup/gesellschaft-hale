package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (app *Application) initServer() {
	for range time.Tick(2 * time.Second) {
		list := app.GetClientsList()
		var clients []Client

		for _, c := range list {
			clients = append(clients, *c)
		}

		log.Printf("clients connected: %+v", clients)
	}
}

func (s *Server) StatusChan(stream protocol.Haler_StatusChanServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream has ended")
		default:
			status, err := stream.Recv()
			if err != nil {
				fmt.Printf("got error while recv: %v", err)
				continue
			}

			current, exists := s.app.Clients.Load(status.GetId())
			// if not existing, we create it fully
			if !exists {
				s.app.Clients.Store(status.GetId(), &Client{
					id:    status.GetId(),
					slots: status.GetSlots(),
				})
				continue
			}

			current.(*Client).slots = status.GetSlots()
			s.app.Clients.Store(status.GetId(), current)
		}
	}
}

func (s *Server) ListenJobs(stream protocol.Haler_ListenJobsServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream has ended")
		default:
			if _, err := stream.Recv(); err != nil {
				log.Printf("err while recv in listen jobs: %v", err)
			}

			req := <-s.app.RequestCh
			if err := stream.Send(req); err != nil {
				log.Printf("err sending job request: %v", err)
			}
		}
	}
}

func (s *Server) SendResult(ctx context.Context, in *protocol.ResponseJobWrapper) (*protocol.Empty, error) {
	s.app.Results.Store(in.GetRequestId(), in)
	return &protocol.Empty{}, nil
}

func (s *Server) HandleExit(ctx context.Context, in *protocol.ExitRequest) (*protocol.Empty, error) {
	s.app.Clients.Delete(in.GetId())
	return &protocol.Empty{}, nil
}
