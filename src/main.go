package main

import (
	"flag"
	"fmt"
	"github.com/jar3b/concron/src/tasks"
	"github.com/jar3b/logrus-levelpad-formatter"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

func healthHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func initLog(debug bool) {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&levelpad.Formatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		LogFormat:       "[%lvl%][%time%] %msg%\n",
		LevelPad:        8,
	})

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	// command arguments
	port := flag.Int("p", 8080, "HTTP server port")
	debug := flag.Bool("debug", false, "debug mode")
	configFile := flag.String("c", "tasks.yaml", "config file location")
	flag.Parse()

	initLog(*debug)

	// get allowed tasks
	var allowedTasks map[string]bool
	if os.Getenv("ALLOWED_TASKS") != "" {
		allowedTasks = make(map[string]bool, 0)
		for _, t := range strings.Split(os.Getenv("ALLOWED_TASKS"), ",") {
			allowedTasks[t] = true
		}
	}

	// manage tasks
	taskList, err := tasks.LoadTasks(*configFile, &allowedTasks)
	if err != nil {
		log.Fatalf("cannot load %s: %v", *configFile, err)
		return
	}

	// start scheduler
	sched, err := tasks.NewScheduler()
	if err != nil {
		log.Fatalf("cannot initialize scheduler: %v", err)
		return
	}
	if err = sched.AddTasks(taskList.Tasks); err != nil {
		log.Fatalf("cannot add task list to scheduler: %v", err)
		return
	}
	if err = sched.Start(); err != nil {
		log.Fatalf("cannot start scheduler: %v", err)
		return
	}

	// setup http router
	router := httprouter.New()
	router.GET("/healthz", healthHandler)

	// start HTTP server
	log.Infof("concron was started on :%d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	if err != nil {
		log.Errorf("cannot start concron http server: %v", err)
	}
}
