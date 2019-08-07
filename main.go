package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/adnilote/stock-proxy/proxy"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
)

const (
	// DSN sentry error monitoring
	DSN string = "https://91d94d4b63c0459cba56427529cc9a09@sentry.io/1519981"
)

// NewLogger initiates zap.logger, which send log to logs/filename
// and stdout
func NewLogger(outputPath []string) (*zap.Logger, error) {
	for _, path := range outputPath {
		if path != "stdout" {
			os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		}
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = outputPath
	return cfg.Build()
}

var lg *zap.Logger

var addr = flag.String("listen-address", ":8082",
	"The address to listen on for HTTP requests.")
var dbaddr = flag.String("mongo-address", "mongo:27017", // mongo or 127.0.0.1
	"The address to connect to mongo.")

func main() {
	flag.Parse()

	// init zap logger
	var err error
	lg, err = NewLogger([]string{
		"proxy.log",
		"stdout",
	})
	if err != nil {
		CaptureError(err, sentry.LevelFatal)
		log.Fatalf("zap.NewDevelopment() failed, error: %v.", err)
	}

	// init sentry for errors
	ConfigureSentry(DSN)

	// register monitoring
	reqLeft := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "requests_left"})
	prometheus.MustRegister(reqLeft)

	// connect to db
	sess, err := mgo.Dial("mongodb://" + *dbaddr) 
	if err != nil {
		CaptureError(err, sentry.LevelFatal)
	}

	collection := sess.DB("av").C("timeseries")

	// get amount of records in db
	n, err := collection.Count()
	if err != nil {
		CaptureError(err, sentry.LevelError)
	}
	lg.Info("Start db", zap.Int("collection_count", n))

	// handler
	handler, err := proxy.NewProxy(collection, lg, reqLeft)
	if err != nil {
		CaptureError(err, sentry.LevelFatal)
	}
	http.HandleFunc("/sync/", handler.GetOHLCVSync)
	http.HandleFunc("/async/", handler.GetOHLCVAsync)
	http.HandleFunc("/history/", handler.GetHistory)
	http.Handle("/health", promhttp.Handler())

	lg.Info("starting server at :8082")

	if http.ListenAndServe(*addr, nil) != nil { //192.168.1.254
		CaptureError(err, sentry.LevelFatal)
	}
}
