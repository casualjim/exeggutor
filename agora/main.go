package main

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/gocraft/web"
	"github.com/imdario/mergo"
	"github.com/jessevdk/go-flags"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/agora/api"
	"github.com/reverb/exeggutor/agora/middlewares"
)

var log = logging.MustGetLogger("exeggutor.main")

var config exeggutor.Config

func init() {
	config = readConfig()
	setupLogging()
}

func main() {
	// scheduler.Start(scheduler.SchedulerConfig{ZookeeperUrl: config.ZookeeperUrl, MesosMaster: config.MesosMaster, DataDirectory: config.DataDirectory})

	apiRouter := web.New(api.ApplicationsContext{}).Middleware(middlewares.RequestTiming)
	apiRouter.Get("/", (*api.ApplicationsContext).ListAll)

	http.Handle("/api", apiRouter)
	http.Handle("/", http.FileServer(http.Dir("./static/build")))
	trapExit()
	log.Notice("Starting agora at %s:%v", config.Interface, config.Port)
	http.ListenAndServe(fmt.Sprintf("%s:%v", config.Interface, config.Port), nil) // Start the server!
}

func trapExit() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		log.Debug("Stopping because %v", sig)
		// scheduler.Stop()
		log.Notice("Stopped agora application")
		os.Exit(0)
	}()

}

func readConfig() exeggutor.Config {
	var cfg exeggutor.Config
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
	fmt.Printf("the config:\n%+v\n", cfg)
	fmt.Println("Loading json config now")

	d, err := os.Open(cfg.ConfigDirectory + "/application.json")
	if err != nil {
		log.Fatalf("Couldn't read json config at %s/application.json", cfg.ConfigDirectory)
		os.Exit(1)
	}
	defer d.Close()
	dec := json.NewDecoder(d)
	var jcfg exeggutor.Config
	dec.Decode(&jcfg)

	mergo.Merge(&cfg, jcfg)
	return cfg
}

func setupLogging() {
	// Customize the output format
	logging.SetFormatter(logging.MustStringFormatter("%{level} %{message}"))

	// Setup one stdout and one syslog backend.
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	if strings.HasPrefix(config.Mode, "prod") {
		logPath := config.LogDirectory + "/agora.log"
		os.MkdirAll(config.LogDirectory, 0755)
		logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Couldn't open log file at %s, because %v", logPath, err)
		}
		defer logFile.Close()
		fileBackend := logging.NewLogBackend(logFile, "", stdlog.LstdFlags|stdlog.Lshortfile)

		logging.SetBackend(logBackend, fileBackend)
	}
}
