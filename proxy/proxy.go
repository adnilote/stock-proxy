package proxy

// url := "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn"

// history not duplicate
import (
	"io/ioutil"
	"net/http"

	av "github.com/adnilote/stock-proxy/av-client"
	counter "github.com/adnilote/stock-proxy/rate-counter"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/getsentry/sentry-go"

	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
)

const (
	// REQLIMIT amount of request per minute
	REQLIMIT int64 = 5
	// APIKey for Alpha Vantage
	APIKey string = "7Z29L509PNF9IE24"
)

// Proxy which limits clients request by REQLIMIT
type Proxy struct {
	db       *MongoDB
	av       *av.AvClient
	counter  *counter.Counter
	throttle chan struct{}
	lg       *zap.Logger
	reqLeft  prometheus.Gauge
}

// NewProxy return proxy instance. Required mongoDB collection db
// and zap logger.
func NewProxy(db *mgo.Collection, lg *zap.Logger, reqLeft prometheus.Gauge) (*Proxy, error) {

	h := &Proxy{
		db:       &MongoDB{db: db},
		av:       av.NewAvClient(APIKey),
		counter:  counter.NewCounter(60),
		throttle: make(chan struct{}),
		lg:       lg,
		reqLeft:  reqLeft,
	}

	// limits requests
	go func() {
		for {
			if h.counter.Rate() < REQLIMIT {
				h.throttle <- struct{}{}
				h.counter.Incr()
				reqLeft.Set(float64(5 - h.counter.Rate()))
			}
		}
	}()

	return h, nil
}

func (p *Proxy) getOHLCV(w http.ResponseWriter, url, ticker string) {

	// send request to server
	resp, err := p.av.Conn.Get(url)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		CaptureError(err, sentry.LevelError, p.lg, map[string]interface{}{"counter": p.counter.Rate()})
		return
	}
	// read response
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		CaptureError(err, sentry.LevelError, p.lg)
		return
	}

	// add record to db
	err = p.db.Add(ticker, respBody)
	if err != nil {
		CaptureError(err, sentry.LevelError, p.lg, map[string]interface{}{"ticker": ticker})
	}

	// send response to client
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(respBody)
	p.lg.Info("send ticker to client", zap.String("ticker", ticker), zap.Int("counter", int(p.counter.Rate())))

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
	// block if exceed limit
	<-p.throttle

	// handle request
	p.getOHLCV(w, path, r.URL.Query().Get(av.QuerySymbol))
}

func (p *Proxy) try() bool {
	select {
	case <-p.throttle:
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

	// check if request amount in last 1 min is not exceed limit
	if !p.try() {
		http.Error(w, "Request decline. Limit excedeed.", http.StatusTooManyRequests)
		p.lg.Info("Limit excedeed.", zap.Int("count", int(p.counter.Rate())))
		return
	}

	// handle request
	p.getOHLCV(w, path, r.URL.Query().Get(av.QuerySymbol))
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
