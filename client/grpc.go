package main

import (
	"context"
	"io"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientWrapper struct {
	Id             int32
	Client         protocol.HalerClient
	Conn           *grpc.ClientConn
	cfg            Config
	pool           *Pool
	engine         *Engine
	isReconnecting atomic.Bool
}

func (app *Application) initClient() error {
	id := protocol.GenerateId()
	conn, err := grpc.Dial("localhost:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error dialing the grpc server: %v", err)
	}

	app.Client.Id = id
	app.Client.Client = protocol.NewHalerClient(conn)
	app.Client.Conn = conn

	// we dn't wanna continue until connected to the server properly
	for range time.Tick(100 * time.Millisecond) {
		state := conn.GetState()
		if state.String() == "READY" {
			log.Println("connected to the server.")
			break
		}
	}

	go app.statusLoop()
	go app.listenLoop()
	return nil
}

func (app *Application) handleConnectivity(err error) {
	if err == io.EOF || err == grpc.ErrClientConnTimeout || err == grpc.ErrClientConnClosing || strings.Contains(err.Error(), "refused") {
		if !app.Client.isReconnecting.Load() {
			log.Println("connection type error, trying to reconnect")
		}
	}
	log.Printf("other connectivity error: %v", err)
	return
}

func (app *Application) statusLoop() {
	for range time.Tick(time.Second) {
		stream, err := app.Client.Client.StatusChan(context.Background())
		if err != nil {
			app.handleConnectivity(err)
			continue
		}

		slots := app.currentCapacity()
		if err := stream.Send(&protocol.Status{
			Id:    app.Client.Id,
			Slots: slots,
		}); err != nil {
			if err == io.EOF {
				break
			}

			log.Printf("err sending status: %v", err)
		}
	}
}

func (app *Application) listenLoop() {
	for range time.Tick(time.Second) {
		stream, err := app.Client.Client.ListenJobs(context.Background())
		if err != nil {
			app.handleConnectivity(err)
			continue
		}

		if err := stream.Send(&protocol.Empty{}); err != nil {
			log.Printf("err sending in listenLoop: %v", err)
			continue
		}

		if job, err := stream.Recv(); err == nil {
			if job.GetClientId() == app.Client.Id {
				go app.Client.handleJobRequest(job)
			}
		}

	}

}

func (cw *ClientWrapper) handleJobRequest(j *protocol.RequestJobWrapper) {
	switch j.GetType() {
	case protocol.MessageType_UNSPECIFIED:
		cw.handleJobError(j, protocol.ErrUnspecifiedRequestType)
	case protocol.MessageType_GET_MAILS:
		cw.handleJobGetMail(j)
	case protocol.MessageType_GET_KEYWORD:
		cw.handleJobGetKeyword(j)
	default:
		cw.handleJobError(j, protocol.ErrUnknownRequestType)
	}
}

func (cw *ClientWrapper) handleJobError(j *protocol.RequestJobWrapper, err error) {
	cw.Client.SendResult(context.Background(), &protocol.ResponseJobWrapper{
		RequestId: j.GetRequestId(),
		Type:      protocol.MessageType_ERROR,
		Error:     err.Error(),
	})
}

func (cw *ClientWrapper) handleJobGetMail(j *protocol.RequestJobWrapper) {
	jobs := makeJobsFromUrls(j.GetUrls(), actionExtractMails)
	results, err := cw.smartLaunch(jobs)
	if err != nil {
		cw.handleJobError(j, err)
		return
	}

	cw.Client.SendResult(context.Background(), &protocol.ResponseJobWrapper{
		RequestId: j.GetRequestId(),
		Type:      protocol.MessageType_GET_MAILS,
		Result:    results,
	})
}

func (cw *ClientWrapper) handleJobGetKeyword(j *protocol.RequestJobWrapper) {
	results, err := cw.engine.scrapeKeyword(j.GetKeyword(), int(j.GetPagesCount()))
	if err != nil {
		cw.handleJobError(j, err)
		return
	}

	cw.Client.SendResult(context.Background(), &protocol.ResponseJobWrapper{
		RequestId: j.GetRequestId(),
		Type:      protocol.MessageType_GET_KEYWORD,
		Result:    results,
	})
}
