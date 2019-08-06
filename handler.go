package main

// url := "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn"

// log
// history not duplicate
import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/getsentry/sentry-go"

	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
)

type Proxy struct {
	db       *MongoDB
	av       *AvClient
	counter  *Counter
	reqLimit int64
	throttle chan struct{}
}

func NewProxy(apiKey string, db *mgo.Collection, reqLimit int64) (*Proxy, error) { //db *sql.DB

	h := &Proxy{
		db:       &MongoDB{db: db},
		av:       NewAvClient(apiKey),
		counter:  NewCounter(60),
		reqLimit: reqLimit,
		throttle: make(chan struct{}),
	}

	// limits requests
	go func() {
		for {
			if h.counter.Rate() < h.reqLimit {
				h.throttle <- struct{}{}
				h.counter.Incr()
			}
		}
	}()

	return h, nil
}

func (p *Proxy) GetOHLCV(w http.ResponseWriter, url, ticker string) {

	// send request to server
	// resp, err := p.av.Get(url)
	resp, err := p.av.Conn.Get(url)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		CaptureError(err, sentry.LevelError, map[string]interface{}{"counter": p.counter.Rate()})
		// lg.Error("Error loading intraday series from server",
		// 	zap.Error(err),
		// 	zap.Int("counter", int(p.counter.Rate())),
		// )
		return
	}
	fmt.Println(resp.Header)
	// read response
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		CaptureError(err, sentry.LevelError)
		// lg.Error("Error reading response from server", zap.Error(err))
		return
	}

	// add record to db
	err = p.db.Add(ticker, respBody)
	if err != nil {
		CaptureError(err, sentry.LevelError, map[string]interface{}{"ticker": ticker})
	}

	// send response to client
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(respBody)
	lg.Info("send ticker to client", zap.String("ticker", ticker), zap.Int("counter", int(p.counter.Rate())))

}

// TimeSeriesSync sends ohlcv in minute syncroniously,
// i.e. client blocks until he got response
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
	p.GetOHLCV(w, path, r.URL.Query().Get(QuerySymbol))
}

func (h *Proxy) try() bool {
	select {
	case <-h.throttle:
		return true
	default:
		return false
	}
}

// TimeSeriesAsync immediately returns response, if limit not exceeded,
// otherwise returns "TooManyRequests" and close connection.
func (p *Proxy) GetOHLCVAsync(w http.ResponseWriter, r *http.Request) {
	path, err := p.av.URL(r.URL.Query())
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// check if request amount in last 1 min is not exceed limit
	if !p.try() {
		http.Error(w, "Request decline. Limit excedeed.", http.StatusTooManyRequests)
		lg.Info("Limit excedeed.", zap.Int("count", int(p.counter.Rate())))
		return
	}

	// handle request
	p.GetOHLCV(w, path, r.URL.Query().Get(QuerySymbol))
}

// History returns requests by ticker
func (h *Proxy) GetHistory(w http.ResponseWriter, r *http.Request) {

	ticker := r.URL.Query().Get(QuerySymbol)
	if ticker == "" {
		http.Error(w, "ticker required", http.StatusBadRequest)
		return
	}

	// Get record from db
	item, err := h.db.Get(ticker)
	if err != nil {
		if err == mgo.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("No history for " + ticker))
			lg.Info("Return no history", zap.Error(err), zap.String("ticker", ticker))
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			// lg.Error("Return err to client", zap.Error(err), zap.String("ticker", ticker))
			CaptureError(err, sentry.LevelError, map[string]interface{}{"ticker": ticker})
		}
		return
	}

	// Send history to client
	w.Header().Set("Content-Type", "application/json")
	for _, i := range item.Ohlcv {
		w.Write(i.Data)
	}
	w.WriteHeader(http.StatusOK)

	// lg.Debug("Return history", zap.String("ticker", ticker))
}
