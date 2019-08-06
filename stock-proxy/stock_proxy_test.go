package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"strings"

	av "github.com/adnilote/stock-proxy/av-client"

	mgo "gopkg.in/mgo.v2"
)

var handler *Proxy

const (
	proxyAdd string = "http://127.0.0.1:8082"
	dbURL    string = "mongodb://mongo:27010"
)

var mimeType map[string]string

type TestCaseM struct {
	params     map[string]string
	statusCode int
	info       string
	url        string
	historyURL string
}

func init() {

	mimeType = map[string]string{
		"json": "application/json",
		"csv":  "application/x-download",
	}
	// init zap logger
	var err error
	lg, err := NewLogger([]string{
		"proxy.log",
	})
	if err != nil {

		log.Fatalf("zap.NewDevelopment() failed, error: %v.", err)
	}

	// init sentry for errors

	// connect to db
	sess, err := mgo.Dial(dbURL)
	if err != nil {
		log.Fatalf("Error connecting to mongoDB: %v", err)
	}

	collection := sess.DB("av").C("timeseries")

	// get amount of records in db
	n, err := collection.Count()
	if err != nil {
		log.Fatalf("Error count in mongoDB: %v", err)
	}
	log.Printf("Start db collection_count = %d", n)

	// handler
	handler, err = NewProxy(collection, lg)
	if err != nil {
		log.Fatalf("Error in NewProxy: %v", err)
	}
}

func TestGetOHLCV(t *testing.T) {

	cases := []TestCaseM{
		TestCaseM{
			params: map[string]string{
				av.QueryFunction: "TIME_SERIES_DAILY_ADJUSTED",
				av.QuerySymbol:   "amzn",
			},
			info: "Daily Time Series with Splits and Dividend Events",
		},
		TestCaseM{
			params: map[string]string{
				av.QueryFunction: "TIME_SERIES_DAILY_ADJUSTED",
				av.QuerySymbol:   "amzn",
				av.QueryDataType: "csv",
			},
		},
		TestCaseM{
			params: map[string]string{
				av.QueryFunction: "TIME_SERIES_INTRADAY",
				av.QueryInterval: "5min",
				av.QuerySymbol:   "amzn",
			},
			info: "Intraday (5min) open, high, low, close prices and volume",
		},
		TestCaseM{
			params: map[string]string{
				av.QueryFunction:   "TIME_SERIES_INTRADAY",
				av.QueryInterval:   "1min",
				av.QuerySymbol:     "msft",
				av.QueryOutputSize: "full",
			},
			info: "Intraday (1min) open, high, low, close prices and volume",
		},
		TestCaseM{
			params: map[string]string{
				av.QueryFunction: "TIME_SERIES_INTRADAY",
				av.QuerySymbol:   "amzn",
			},
			statusCode: http.StatusBadRequest,
		},
		TestCaseM{
			params: map[string]string{
				av.QueryFunction: "TIME_SERIES_DAILY_ADJUSTED",
			},
			statusCode: http.StatusBadRequest,
		},
	}

	for caseNum, cs := range cases {

		u, _ := url.ParseRequestURI(proxyAdd)
		u.Path = "/sync/"

		params := url.Values{}
		for key, val := range cs.params {
			params.Set(key, val)
		}
		u.RawQuery = params.Encode()

		req, _ := http.NewRequest("GET", u.String(), nil)
		w := httptest.NewRecorder()

		handler.GetOHLCVSync(w, req)

		if cs.statusCode == 0 {
			cs.statusCode = http.StatusOK
		}
		if w.Code != cs.statusCode {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				caseNum, w.Code, cs.statusCode)
		}

		if w.Code != http.StatusOK {
			break
		}

		if _, ok := cs.params[av.QueryDataType]; !ok {
			cs.params[av.QueryDataType] = "json"
		}

		gotContentType := w.Header().Get("Content-Type")
		if gotContentType != mimeType[cs.params[av.QueryDataType]] {
			t.Fatalf("[%d] wrong Content-Type: got %s, expected %s",
				caseNum, gotContentType, mimeType[cs.params[av.QueryDataType]])
		}

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		var item map[string]interface{}
		if gotContentType == "application/json" {
			err := json.Unmarshal(body, &item)
			if err != nil {
				t.Errorf("[%d] %s", caseNum, err)
			}

			if _, ok := item["Meta Data"]; !ok {
				t.Fatalf("[%d] Expected Metadata in response: %v", caseNum, item)
			}
			if cs.info != "" {
				info := item["Meta Data"].(map[string]interface{})["1. Information"].(string)
				if info != cs.info {
					t.Fatalf("[%d] wrong Meta Data information: got %s, expected %s",
						caseNum, info, cs.info)
				}
			}

			symbol := item["Meta Data"].(map[string]interface{})["2. Symbol"].(string)
			if symbol != cs.params[av.QuerySymbol] {
				t.Errorf("[%d] wrong Meta Data Symbol: got %s, expected %s",
					caseNum, symbol, cs.params[av.QuerySymbol])
			}

			if item["Meta Data"].(map[string]interface{})["4. Interval"] != nil {
				interval := item["Meta Data"].(map[string]interface{})["4. Interval"].(string)
				if _, ok := cs.params[av.QueryInterval]; ok {
					if interval != cs.params[av.QueryInterval] {
						t.Errorf("[%d] wrong Meta Data Interval: got %s, expected %s",
							caseNum, interval, cs.params[av.QueryInterval])
					}
				}
			}

			if item["Meta Data"].(map[string]interface{})["5. Output Size"] != nil {
				size := item["Meta Data"].(map[string]interface{})["5. Output Size"].(string)
				if _, ok := cs.params[av.QueryOutputSize]; ok {
					if strings.Contains(size, cs.params[av.QueryInterval]) {
						t.Errorf("[%d] wrong Meta Data Output Size: got %s, expected %s",
							caseNum, size, cs.params[av.QueryOutputSize])
					}
				}
			}

		}

	}

}

func TestGetOHLCVSyncLimit(t *testing.T) {
	// to be sure that, tests will pass
	time.Sleep(time.Minute)

	params := url.Values{}
	params.Set(av.QueryFunction, "TIME_SERIES_DAILY_ADJUSTED")
	params.Set(av.QuerySymbol, "amzn")
	u, _ := url.ParseRequestURI(proxyAdd)
	u.Path = "/sync/"
	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Errorf("error sending request: %s", err)
	}

	for i := 0; i < 7; i++ {
		w := httptest.NewRecorder()
		handler.GetOHLCVSync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				i, w.Code, http.StatusOK)
		}

	}

}

// TestGetOHLCVASyncLimit test that we can't exceed limit.
// Requires more than 1 minute to work and that in minute there
// were not any requests.
func TestGetOHLCVAsyncLimit(t *testing.T) {
	// to be sure that, tests will pass
	time.Sleep(time.Minute)

	params := url.Values{}
	params.Set(av.QueryFunction, "TIME_SERIES_DAILY_ADJUSTED")
	params.Set(av.QuerySymbol, "amzn")
	u, _ := url.ParseRequestURI(proxyAdd)
	u.Path = "/async/"
	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Errorf("error sending request: %s", err)
	}

	// First 5 requests must be ok
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.GetOHLCVAsync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				i, w.Code, http.StatusOK)
		}
	}

	// After 5 request, exceed limit
	w := httptest.NewRecorder()
	handler.GetOHLCVAsync(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("[5] wrong StatusCode: got %d, expected %d",
			w.Code, http.StatusTooManyRequests)
	}

	// after 1 minute, must be ok again
	time.Sleep(time.Minute)
	for i := 6; i < 8; i++ {
		w := httptest.NewRecorder()
		handler.GetOHLCVSync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				i, w.Code, http.StatusOK)
		}
	}
}
func TestGetHistory(t *testing.T) {
	// to be sure that, tests will pass
	// time.Sleep(time.Minute)

	cases := []TestCaseM{
		TestCaseM{
			url:        proxyAdd + "/query?function=TIME_SERIES_INTRADAY&interval=1min&symbol=amzn",
			statusCode: http.StatusOK,
			historyURL: "http://127.0.0.1:8082/history/?symbol=amzn",
		},
		TestCaseM{
			url:        proxyAdd + "/query?function=TIME_SERIES_INTRADAY&interval=1min&symbol=amzn",
			statusCode: http.StatusBadRequest,
			historyURL: "http://127.0.0.1:8082/history/",
		},
		TestCaseM{
			url:        proxyAdd + "/query?function=TIME_SERIES_INTRADAY&interval=1min&symbol=amzn",
			statusCode: http.StatusNotFound,
			historyURL: "http://127.0.0.1:8082/history/?symbol=asd",
		},
		TestCaseM{
			url:        proxyAdd + "/query?function=TIME_SERIES_INTRADAY&interval=1min&symbol=AMV",
			statusCode: http.StatusOK,
			historyURL: "http://127.0.0.1:8082/history/?symbol=AMV",
		},
	}
	for caseNum, cs := range cases {
		req, err := http.NewRequest("GET", cs.url, nil)
		if err != nil {
			t.Errorf("[%d] error sending request: %s", caseNum, err)
		}
		w := httptest.NewRecorder()

		handler.GetOHLCVSync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				caseNum, w.Code, http.StatusOK)
		}

		req, err = http.NewRequest("GET", cs.historyURL, nil)
		if err != nil {
			t.Errorf("[%d] error sending request: %s", caseNum, err)
		}
		w = httptest.NewRecorder()

		handler.GetHistory(w, req)
		if w.Code != cs.statusCode {
			t.Errorf("[%d] wrong StatusCode: got %d, expected %d",
				caseNum, w.Code, cs.statusCode)
		}

	}

}
