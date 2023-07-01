package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Pool struct {
	idle     chan *Browser
	active   chan *Browser
	capacity int
	stats    PoolStats
}

type PoolStats struct {
	idleCount   atomic.Int32
	activeCount atomic.Int32
}

func (app *Application) startPool(capacity int) {
	if capacity == 0 {
		log.Fatal(protocol.ErrPoolZeroCap)
	}

	app.Client.pool = &Pool{
		idle:     make(chan *Browser, capacity),
		active:   make(chan *Browser, capacity),
		capacity: capacity,
	}

	for i := 0; i < capacity; i++ {
		app.Client.pool.stats.idleCount.Add(1)
		app.Client.pool.idle <- app.newBrowser(i, app.Client.cfg.browser.timeout)
	}
}

func (p *Pool) loan() (*Browser, error) {
	for {
		select {
		case browser := <-p.idle:
			p.stats.idleCount.Add(-1)
			p.stats.activeCount.Add(1)
			browser.active = true
			fmt.Println("loaning browser", browser.id)
			return browser, nil
		case <-time.After(2 * time.Second):
			return nil, protocol.ErrNoBrowserAvailable
		}
	}
}

func (p *Pool) put(b *Browser) {
	if b.active == false {
		fmt.Println(protocol.ErrBrowserNotActive)
		return
	}

	fmt.Println("returning the browser", b.id)
	b.cleanup()
	p.stats.idleCount.Add(1)
	p.stats.activeCount.Add(-1)
	p.idle <- b
}

func (cw *ClientWrapper) hasCapacity(numTasks int) bool {
	currentCapacity := int(cw.pool.stats.idleCount.Load() * cw.cfg.queue.maxTasks)
	return numTasks <= currentCapacity
}

func (cw *ClientWrapper) smartLaunch(jobs []*Job) ([]*protocol.Website, error) {
	numJobs := len(jobs)
	maxTasks := int(cw.cfg.queue.maxTasks)
	if hasCapacity := cw.hasCapacity(numJobs); !hasCapacity {
		return nil, protocol.ErrNotEnoughCapacity
	}

	var results Results
	var wg sync.WaitGroup

	for i := 0; i < numJobs; i += maxTasks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			end := i + maxTasks
			if end > numJobs {
				end = numJobs
			}

			b, err := cw.pool.loan()
			if err != nil {
				return
			}
			defer cw.pool.put(b)

			go func() {
				res := b.processQueue(jobs[i:end]...)
				results.Append(res...)
			}()

		}(i)
	}
	wg.Wait()

	return results.Get(), nil
}

type Results struct {
	websites []*protocol.Website
	m        sync.Mutex
}

func (r *Results) Append(website ...*protocol.Website) {
	r.m.Lock()
	defer r.m.Unlock()
	r.websites = append(r.websites, website...)
}

func (r *Results) Get() []*protocol.Website {
	r.m.Lock()
	defer r.m.Unlock()

	copySlice := make([]*protocol.Website, len(r.websites))
	copy(copySlice, r.websites)
	return copySlice
}

func (r *Results) ReadLen() int {
	r.m.Lock()
	defer r.m.Unlock()

	return len(r.websites)
}

func (app *Application) getFreeSlots() int32 {
	return app.Client.pool.stats.idleCount.Load() * app.Client.cfg.queue.maxTasks
}
