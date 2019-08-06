package main

// import (
// 	"log"
// 	"net/http"
// 	"os"

// 	"github.com/getsentry/sentry-go"
// 	"github.com/prometheus/client_golang/prometheus/promhttp"
// 	"go.uber.org/zap"
// 	mgo "gopkg.in/mgo.v2"
// )

// const (
// 	ApiKey   string = "7Z29L509PNF9IE24"
// 	REQLIMIT int64  = 5
// 	DSN      string = "https://91d94d4b63c0459cba56427529cc9a09@sentry.io/1519981"
// 	DbURL    string = "mongodb://mongo:27017" //mongo
// )

// // NewLogger initiates zap.logger, which send log to logs/filename
// // and stdout
// func NewLogger(outputPath []string) (*zap.Logger, error) {
// 	for _, path := range outputPath {
// 		if path != "stdout" {
// 			os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
// 		}
// 	}

// 	cfg := zap.NewDevelopmentConfig()
// 	cfg.OutputPaths = outputPath
// 	return cfg.Build()
// }

// var lg *zap.Logger

// func main() {

// 	// init zap logger
// 	var err error
// 	lg, err = NewLogger([]string{
// 		"proxy.log",
// 		"stdout",
// 	})
// 	if err != nil {
// 		CaptureError(err, sentry.LevelFatal)
// 		log.Fatalf("zap.NewDevelopment() failed, error: %v.", err)
// 	}

// 	// init sentry for errors
// 	ConfigureSentry(DSN)

// 	// connect to db
// 	sess, err := mgo.Dial(DbURL)
// 	if err != nil {
// 		CaptureError(err, sentry.LevelFatal)
// 	}

// 	collection := sess.DB("av").C("timeseries")

// 	// get amount of records in db
// 	n, err := collection.Count()
// 	if err != nil {
// 		CaptureError(err, sentry.LevelError)
// 	}
// 	lg.Info("Start db", zap.Int("collection_count", n))

// 	// handler
// 	handler, err := NewProxy(ApiKey, collection, REQLIMIT)
// 	if err != nil {
// 		CaptureError(err, sentry.LevelFatal)
// 	}
// 	http.HandleFunc("/sync/", handler.GetOHLCVSync)
// 	http.HandleFunc("/async/", handler.GetOHLCVAsync)
// 	http.HandleFunc("/history/", handler.GetHistory)
// 	http.Handle("/health", promhttp.Handler())

// 	lg.Info("starting server at :8082")

// 	if http.ListenAndServe(":8082", nil) != nil { //192.168.1.254
// 		CaptureError(err, sentry.LevelFatal)
// 	}
// }
