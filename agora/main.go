package main

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/imdario/mergo"
	"github.com/jessevdk/go-flags"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/agora/api"
	"github.com/reverb/exeggutor/agora/middlewares"
	"github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/store"
)

var log = logging.MustGetLogger("exeggutor.main")

var (
	config  exeggutor.Config
	context api.APIContext
)

func init() {
	config = readConfig()
	context = api.APIContext{Config: &config}
	setupLogging()
}

func main() {

	scheduler.Start(config)
	appStore, err := store.NewMdbStore(config.DataDirectory + "/applications")
	if err != nil {
		log.Fatalf("Couldn't initialize app database at %s/applications, because %v", config.DataDirectory, err)
	}
	context.FrameworkIDState = scheduler.FrameworkIDState
	context.AppStore = appStore

	applicationsController := api.NewApplicationsController(&context)
	applicationsController.Start()
	mesosController := api.NewMesosController()

	router := httprouter.New()
	router.GET("/api/applications", applicationsController.ListAll)
	router.GET("/api/applications/:name", applicationsController.ShowOne)
	router.POST("/api/applications", applicationsController.Save)
	router.PUT("/api/applications/:name", applicationsController.Save)
	router.DELETE("/api/applications/:name", applicationsController.Delete)

	router.GET("/api/mesos/fwid", mesosController.ShowFrameworkID)

	n := negroni.New()
	n.Use(middlewares.NewJSONOnlyAPI())
	n.Use(middlewares.NewRecovery())
	n.Use(middlewares.NewLogger())
	n.Use(negroni.NewStatic(http.Dir("static/build")))
	n.UseHandler(router)

	trapExit(func() {
		scheduler.Stop()
		appStore.Stop()
	})

	n.Run(fmt.Sprintf("%s:%v", config.Interface, config.Port))
}

func trapExit(onClose func()) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		log.Debug("Stopping because %v", sig)
		onClose()
		log.Notice("Stopped agora application")
		os.Exit(0)
	}()

}

func readConfig() exeggutor.Config {
	var cfg exeggutor.Config
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}

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

	envPort := os.Getenv("PORT")
	if envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err != nil {
			log.Fatalf("The value of the port environment variable is %v which is not convertible to int", envPort)
		}
		cfg.Port = p
	}

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
