package main

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gocraft/web"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/agora/middlewares"
	"github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/state"

	"github.com/imdario/mergo"
	"github.com/jessevdk/go-flags"
)

var log = logging.MustGetLogger("exeggutor.main")

type Context struct {
	FrameworkIDState *state.FrameworkIDState
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

// FrameworkID returns a json structure for the framework id of this application
func (m *Context) ShowFrameworkID(rw web.ResponseWriter, req *web.Request) {
	state := scheduler.FrameworkIDState.Get()
	id := state.GetValue()
	enc := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	enc.Encode(&fwID{Value: &id})
}

var Config exeggutor.Config

func init() {
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
	Config = cfg
}

func main() {

	// Customize the output format
	logging.SetFormatter(logging.MustStringFormatter("%{level} %{message}"))

	// Setup one stdout and one syslog backend.
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true

	logPath := Config.LogDirectory + "/agora.log"
	os.MkdirAll(Config.LogDirectory, 0755)
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Couldn't open log file at %s, because %v", logPath, err)
	}
	defer logFile.Close()
	fileBackend := logging.NewLogBackend(logFile, "", stdlog.LstdFlags|stdlog.Lshortfile)

	logging.SetBackend(logBackend, fileBackend)

	scheduler.Start()
	router := web.New(Context{FrameworkIDState: scheduler.FrameworkIDState}). // Create your router
											Middleware(middlewares.RequestLogging). // Use some included middleware
											Middleware(middlewares.RequestTiming).  // Use some included middleware
											Get("/fwid", (*Context).ShowFrameworkID)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		log.Debug("Stopping because %v", sig)
		scheduler.Stop()
		log.Notice("Stopped agora application")
		os.Exit(0)
	}()

	log.Notice("starting agore at localhost:3000")
	http.ListenAndServe("localhost:3000", router) // Start the server!
}
