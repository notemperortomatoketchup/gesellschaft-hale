package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Queue struct {
	running    atomic.Int32
	maxTasks   int32
	maxRunning int32
	incomingCh chan *Job     // We get a Job
	stopCh     chan struct{} // to stop
}

type Job struct {
	id      int
	action  BrowserAction
	website *protocol.Website
}

func (app *Application) newQueue() *Queue {
	return &Queue{
		maxTasks:   app.Client.cfg.queue.maxTasks,
		maxRunning: app.Client.cfg.queue.maxRunning,
		incomingCh: make(chan *Job, 75),
		stopCh:     make(chan struct{}, 1),
	}
}

func makeJobsFromUrls(urls []string, action BrowserAction) []*Job {
	var jobs []*Job
	for i, u := range urls {
		u, _ = getBaseUrl(u)
		jobs = append(
			jobs,
			&Job{id: i, website: &protocol.Website{BaseUrl: u}, action: action},
		)
	}
	return jobs
}

func makeJobsFromWebsites(websites []*protocol.Website, action BrowserAction) []*Job {
	var jobs []*Job
	for i, w := range websites {
		jobs = append(
			jobs,
			&Job{id: i, website: w, action: action},
		)
	}
	return jobs
}

func (q *Queue) add(jobs ...*Job) {
	for _, job := range jobs {
		go func(j *Job) {
			q.incomingCh <- j
		}(job)
	}
}

// here we inherit (b) and not (q) because we need access to it to give to the actions.
func (b *Browser) processQueue(jobs ...*Job) []*protocol.Website {
	var results Results
	b.queue.add(jobs...)

	go func() {
	mainloop:
		for {
			select {
			case job := <-b.queue.incomingCh:
				go func(j *Job) {
					for range time.Tick(time.Second) {
						if b.queue.running.Load() >= b.queue.maxRunning {
							continue
						}
						b.queue.running.Add(1)
						j.action(b, j.website)
						results.Append(j.website)
						b.queue.running.Add(-1)
						break
					}
					return
				}(job)
			case <-b.queue.stopCh:
				fmt.Println("exiting queue.")
				break mainloop
			}
		}
	}()

	for results.ReadLen() != len(jobs) {
		time.Sleep(1 * time.Second)
	}

	b.queue.stopCh <- struct{}{}

	return results.Get()
}
