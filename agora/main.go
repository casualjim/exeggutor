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

	"github.com/antage/eventsource"
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
	"github.com/reverb/exeggutor/tasks"
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

	es := eventsource.New(nil, nil)
	config.EventSource = &es
	mgr, err := tasks.NewDefaultTaskManager(&config)
	if err != nil {
		log.Fatalf("Couldn't initialize the task manager because:%v", err)
	}
	mgr.Start()

	framework := scheduler.NewFramework(&config, mgr)
	err = framework.Start()
	if err != nil {
		log.Fatalf("Couldn't initialize the exeggutor scheduler framework because:%v", err)
	}

	appStore, err := store.NewMdbStore(config.DataDirectory + "/applications")
	if err != nil {
		log.Fatalf("Couldn't initialize app database at %s/applications, because %v", config.DataDirectory, err)
	}
	appStore.Start()

	context.Framework = framework
	context.AppStore = appStore

	applicationsController := api.NewApplicationsController(&context)
	mesosController := api.NewMesosController(&context)

	router := httprouter.New()
	router.GET("/favicon.ico", func(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		rw.WriteHeader(http.StatusNotFound)
	})
	router.GET("/api/applications", applicationsController.ListAll)
	router.GET("/api/applications/:name", applicationsController.ShowOne)
	router.POST("/api/applications", applicationsController.Save)
	router.PUT("/api/applications/:name", applicationsController.Save)
	router.DELETE("/api/applications/:name", applicationsController.Delete)
	router.POST("/api/applications/:name/deploy", applicationsController.Deploy)
	router.GET("/api/mesos/fwid", mesosController.ShowFrameworkID)

	log.Info("serving static files from: %v", config.StaticFiles)
	staticFS := http.Dir(config.StaticFiles)

	router.NotFound = http.FileServer(staticFS).ServeHTTP

	n := negroni.New()

	n.Use(middlewares.NewEventSource(es))
	n.Use(middlewares.NewJSONOnlyAPI())
	n.Use(middlewares.NewRecovery())
	n.Use(middlewares.NewLogger())
	n.Use(negroni.NewStatic(staticFS))
	n.UseHandler(router)

	trapExit(func() {
		mgr.Stop()
		es.Close()
		framework.Stop()
		appStore.Stop()
	})

	addr := fmt.Sprintf("%s:%v", config.Interface, config.Port)
	log.Notice("Starting server at %s.", addr)
	// http.ListenAndServeTLS(addr, "star_helloreverb_com.cer", "helloreverb.key", n)
	http.ListenAndServe(addr, n)
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
