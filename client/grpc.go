package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientWrapper struct {
	Id             uint32
	Client         protocol.HalerClient
	Conn           *grpc.ClientConn
	cfg            Config
	pool           *Pool
	engine         *Engine
	isReconnecting atomic.Bool
}

func (app *Application) resolveDomain() string {
	// start at localhost, in case we are in dev
	// adjust if we are not with the if
	resolved := "localhost"

	if !app.Client.cfg.core.devMode {
		ips, err := net.LookupIP(app.Client.cfg.core.domain)
		if err != nil {
			log.Fatalf("err resolving domain: %v", err)
		}
		resolved = ips[0].String()
	}

	return resolved
}

func (app *Application) initClient() error {
	id := protocol.GenerateId()
	ip := app.resolveDomain()
	fmt.Println("ip ->", ip)

	conn, err := grpc.Dial(ip+":50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry())
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
	log.Printf("other connectivity error: %v\n", err)
	return
}

func (app *Application) statusLoop() {
	stream, err := app.Client.Client.StatusChan(context.Background())
	if err != nil {
		app.handleConnectivity(err)
	}

	for range time.Tick(time.Second) {
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
	stream, err := app.Client.Client.ListenJobs(context.Background())
	if err != nil {
		app.handleConnectivity(err)
	}

	for range time.Tick(time.Second) {
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
	case protocol.MessageType_GET_MAILS_URLS:
		cw.handleJobGetMailUrl(j)
	case protocol.MessageType_GET_MAILS_WEBSITES:
		cw.handleJobGetMailWebsite(j)
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

func (cw *ClientWrapper) handleJobGetMailUrl(j *protocol.RequestJobWrapper) {
	jobs := makeJobsFromUrls(j.GetUrls(), actionExtractMails)

	results, err := cw.smartLaunch(jobs)
	if err != nil {
		cw.handleJobError(j, err)
		return
	}

	_, err = cw.Client.SendResult(context.Background(), &protocol.ResponseJobWrapper{
		RequestId: j.GetRequestId(),
		Type:      protocol.MessageType_GET_MAILS_URLS,
		Result:    results,
	})
}

func (cw *ClientWrapper) handleJobGetMailWebsite(j *protocol.RequestJobWrapper) {
	jobs := makeJobsFromWebsites(j.GetWebsites(), actionExtractMails)
	results, err := cw.smartLaunch(jobs)
	if err != nil {
		cw.handleJobError(j, err)
		return
	}

	cw.Client.SendResult(context.Background(), &protocol.ResponseJobWrapper{
		RequestId: j.GetRequestId(),
		Type:      protocol.MessageType_GET_MAILS_WEBSITES,
		Result:    results,
	})
}

func (cw *ClientWrapper) handleJobGetKeyword(j *protocol.RequestJobWrapper) {
	results, err := cw.engine.scrapeKeyword(j.GetKeyword(), int(j.GetPagesCount()), j.GoogleDomain)
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
