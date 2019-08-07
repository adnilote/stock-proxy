package proxy

// url := "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn"

// history not duplicate
import (
	"io/ioutil"
	"net/http"
	"time"

	av "github.com/adnilote/stock-proxy/av-client"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/getsentry/sentry-go"

	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
)

var (
// MaxQueue = os.Getenv("MAX_QUEUE")
)

const (
	// REQLIMIT amount of request per minute
	REQLIMIT int = 5
	// APIKey for Alpha Vantage
	APIKey string = "7Z29L509PNF9IE24"
	// MaxQueue = max amount of tasks in queque
	MaxQueue int = 10
)

// Proxy which limits clients request by REQLIMIT
type Proxy struct {
	db *MongoDB
	av *av.AvClient

	lg      *zap.Logger
	reqLeft prometheus.Gauge
	tasks   chan *Task
}

// Task is a request from client, which is sent to workers.
type Task struct {
	w      http.ResponseWriter
	url    string
	ticker string
	out    chan struct{}
	// snedClient true - will write response to w
	sendClient bool
}

// NewProxy return proxy instance. Required mongoDB collection db
// and zap logger.
func NewProxy(db *mgo.Collection, lg *zap.Logger, reqLeft prometheus.Gauge) (*Proxy, error) {

	p := &Proxy{
		db:      &MongoDB{db: db},
		av:      av.NewAvClient(APIKey),
		lg:      lg,
		reqLeft: reqLeft,
	}

	p.tasks = make(chan *Task, MaxQueue)

	for i := 0; i < REQLIMIT; i++ {
		go func() {
			for {
				select {
				case task := <-p.tasks:
					p.getOHLCV(task.w, task.url, task.ticker, task.sendClient)
					if task.sendClient {
						task.out <- struct{}{}
					}
					time.Sleep(time.Minute)
				}
			}
		}()
	}

	return p, nil
}

func (p *Proxy) getOHLCV(w http.ResponseWriter, url, ticker string, sendClient bool) {

	// send request to server
	resp, err := p.av.Conn.Get(url)

	if err != nil {
		if sendClient {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		CaptureError(err, sentry.LevelError, p.lg) //map[string]interface{}{"counter": p.counter.Rate()}
		return
	}
	// read response
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if sendClient {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		CaptureError(err, sentry.LevelError, p.lg)
		return
	}

	// add record to db
	err = p.db.Add(ticker, respBody)
	if err != nil {
		CaptureError(err, sentry.LevelError, p.lg, map[string]interface{}{"ticker": ticker})
	}
	p.lg.Debug("Write response to db", zap.String("ticker", ticker))

	// send response to client
	if sendClient {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Write(respBody)
		p.lg.Debug("send ticker to client", zap.String("ticker", ticker)) //zap.Int("counter", int(p.counter.Rate()))
	}

}

// GetOHLCVSync sends ohlcv in minute syncroniously,
// i.e. client blocks until he got response.
//
// Example: http://127.0.0.1:8082/sync/?function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn
func (p *Proxy) GetOHLCVSync(w http.ResponseWriter, r *http.Request) {

	// validate query params and prepare query to go to server
	path, err := p.av.URL(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// send to workers
	task := Task{
		w:          w,
		url:        path,
		ticker:     r.URL.Query().Get(av.QuerySymbol),
		out:        make(chan struct{}),
		sendClient: true,
	}
	// blocks if exceed limit
	p.tasks <- &task

	// wait for request to finish
	<-task.out
}

// try returns true if task queque is not full
func (p *Proxy) try(task *Task) bool {
	select {
	case p.tasks <- task:
		return true
	default:
		return false
	}
}

// GetOHLCVAsync immediately returns response, if limit not exceeded,
// otherwise returns "TooManyRequests" and close connection.
//
// Example: http://127.0.0.1:8082/async/?function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn
func (p *Proxy) GetOHLCVAsync(w http.ResponseWriter, r *http.Request) {
	path, err := p.av.URL(r.URL.Query())
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// send to request workers
	task := Task{
		w:          w,
		url:        path,
		ticker:     r.URL.Query().Get(av.QuerySymbol),
		out:        make(chan struct{}),
		sendClient: false,
	}
	if !p.try(&task) {
		http.Error(w, "Try later", http.StatusTooManyRequests)
		return
	}

	w.Write([]byte("OK"))
}

// GetHistory returns requests by ticker
//
// Example: http://192.168.99.100:8082/history/?symbol=amzn
func (p *Proxy) GetHistory(w http.ResponseWriter, r *http.Request) {

	ticker := r.URL.Query().Get(av.QuerySymbol)
	if ticker == "" {
		http.Error(w, "ticker required", http.StatusBadRequest)
		return
	}

	// Get record from db
	item, err := p.db.Get(ticker)
	if err != nil {
		if err == mgo.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("No history for " + ticker))
			p.lg.Info("Return no history", zap.Error(err), zap.String("ticker", ticker))
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			// p.lg.Error("Return err to client", zap.Error(err), zap.String("ticker", ticker))
			CaptureError(err, sentry.LevelError, p.lg, map[string]interface{}{"ticker": ticker})
		}
		return
	}

	// Send history to client
	w.Header().Set("Content-Type", "application/json")
	for _, i := range item.Ohlcv {
		w.Write(i.Data)
	}
	w.WriteHeader(http.StatusOK)
}
