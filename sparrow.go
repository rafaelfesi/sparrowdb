package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/SparrowDb/sparrowdb/auth"
	"github.com/SparrowDb/sparrowdb/compression"
	"github.com/SparrowDb/sparrowdb/db"
	"github.com/SparrowDb/sparrowdb/http"
	"github.com/SparrowDb/sparrowdb/service"
	"github.com/SparrowDb/sparrowdb/slog"
	"github.com/SparrowDb/sparrowdb/util"
	"github.com/SparrowDb/sparrowdb/web"
)

const (
	// Version SparrowDb version
	Version = "1.0.0"
)

var (
	totalProcs      = runtime.NumCPU()
	configPathFlag  = flag.String("config", "./config/", "Description")
	configProcsFlag = flag.Int("j", totalProcs, "Description")
	instance        = &Instance{}
)

// Instance holds SparrowDb instance configuration
type Instance struct {
	pid            int
	sparrowConfig  *db.SparrowConfig
	databaseConfig *db.DatabaseConfig
	dbManager      *db.DBManager
	httpServer     http.HTTPServer
	httpUI         web.UIServer
	serviceManager service.Manager
}

func checkAndCreateDefaultDirs() {
	dirs := []string{"config", "data", "scripts", "snapshot"}
	for _, val := range dirs {
		if _, err := os.Stat(val); os.IsNotExist(err) {
			util.CreateDir(val)
		}
	}
}

func init() {
	// Sets pid
	instance.pid = os.Getpid()

	createPIDfile()

	slog.SetLogger(slog.NewGlog())
	compression.SetCompressor(compression.NewSnappyCompressor())

	// Configure signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go handleSignal(c)
}

func handleSignal(c chan os.Signal) {
	<-c
	instance.serviceManager.StopAll()
	slog.Infof("Quinting SparrowDB")
	os.Exit(1)
}

func createPIDfile() {
	p := strconv.Itoa(instance.pid)
	ioutil.WriteFile("sparrow.pid", []byte(p), 0644)
}

func main() {
	flag.Parse()

	checkAndCreateDefaultDirs()

	// validate flag processors
	if *configProcsFlag > totalProcs || *configProcsFlag < 0 {
		slog.Fatalf("Invalid number of processors: %d, max of %d", *configProcsFlag, totalProcs)
	}

	slog.Infof("%s v%s", "SparrowDB", Version)
	slog.Infof("PID: %d, Cores: %d", instance.pid, *configProcsFlag)
	runtime.GOMAXPROCS(*configProcsFlag)

	instance.sparrowConfig = db.NewSparrowConfig(*configPathFlag)
	instance.databaseConfig = db.NewDatabaseConfig(*configPathFlag)
	slog.Infof("Database read-only: %v", instance.sparrowConfig.ReadOnly)

	auth.LoadUserConfig(*configPathFlag, instance.sparrowConfig)

	instance.serviceManager = service.NewManager()

	instance.dbManager = db.NewDBManager(instance.sparrowConfig, instance.databaseConfig)
	instance.dbManager.LoadDatabases()
	instance.serviceManager.AddService("dbManager", instance.dbManager)

	instance.httpServer = http.NewHTTPServer(instance.sparrowConfig, instance.dbManager)
	instance.serviceManager.AddService("httpServer", &instance.httpServer)

	if instance.sparrowConfig.EnableWebUI {
		instance.httpUI = web.NewUIServer(instance.sparrowConfig)
		instance.serviceManager.AddService("httpUI", &instance.httpUI)
	}

	instance.serviceManager.StartAll()
}
