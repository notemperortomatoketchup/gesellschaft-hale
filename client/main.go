package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Application struct {
	Client *ClientWrapper
}

type Config struct {
	pool    PoolConfig
	browser BrowserConfig
	queue   QueueConfig
	engine  EngineConfig
	core    CoreConfig
}

type CoreConfig struct {
	devMode bool
	domain  string
}

type PoolConfig struct {
	capacity int
}

type BrowserConfig struct {
	timeout   time.Duration
	noSandbox bool
}

type QueueConfig struct {
	maxTasks   int32
	maxRunning int32
}

type EngineConfig struct {
	baseLink string
}

func main() {
	app := &Application{
		Client: &ClientWrapper{},
	}

	app.loadConfig()
	app.startPool(app.Client.cfg.pool.capacity)
	app.startEngine()
	app.initClient()
	defer app.Client.Conn.Close()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	<-terminate

	app.Client.Client.HandleExit(context.Background(), &protocol.ExitRequest{
		Id: app.Client.Id,
	})
}

func (app *Application) loadConfig() {
	viper.AddConfigPath("./")
	viper.SetConfigFile("config.json")
	viper.SetConfigType("json")
	viper.ReadInConfig()

	app.Client.cfg.core.devMode = viper.GetBool("core.dev_mode")
	app.Client.cfg.core.domain = viper.GetString("core.domain")

	app.Client.cfg.browser.timeout = viper.GetDuration("browser.timeout") * time.Second
	app.Client.cfg.pool.capacity = viper.GetInt("pool.capacity")

	app.Client.cfg.queue.maxTasks = viper.GetInt32("browser.queue.max_tasks")
	app.Client.cfg.queue.maxRunning = viper.GetInt32("browser.queue.max_running")

	app.Client.cfg.engine.baseLink = viper.GetString("engine.base_link")
}
