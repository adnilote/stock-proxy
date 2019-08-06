package av

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// HostDefault is the default host for Alpha Vantage
	HostDefault = "www.alphavantage.co"
)
const (
	schemeHttps = "https"

	QueryApiKey     = "apikey"
	QueryDataType   = "datatype"
	QueryOutputSize = "outputsize"
	QuerySymbol     = "symbol"
	QueryFunction   = "function"
	QueryInterval   = "interval"

	pathQuery = "query"

	requestTimeout = time.Second * 10
)

// Client is a service used to query Alpha Vantage stock data
type AvClient struct {
	Conn   *http.Client
	ApiKey string
}

func NewAvClient(apiKey string) *AvClient {
	Conn := &http.Client{
		Timeout: requestTimeout,
	}

	return &AvClient{
		Conn:   Conn,
		ApiKey: apiKey,
	}
}

// URL validates url values to have required params, such as
// QueryFunction, QueryInterval and QuerySymbol.
// Returns URL to go to alphavantage service.
func (av *AvClient) URL(values url.Values) (string, error) {
	function := values.Get(QueryFunction)
	symbol := values.Get(QuerySymbol)
	if function == "" || symbol == "" {
		return "", errors.New("Params required")
	}

	u := &url.URL{
		Scheme: schemeHttps,
		Host:   HostDefault,
		Path:   pathQuery,
	}

	// base parameters
	query := u.Query()
	query.Set(QueryApiKey, av.ApiKey)
	query.Set(QueryFunction, function)
	query.Set(QuerySymbol, symbol)

	interval := values.Get(QueryInterval)
	if strings.ToUpper(function) == "TIME_SERIES_INTRADAY" {
		if interval == "" {
			return "", errors.New("Params required")
		}
		query.Set(QueryInterval, interval)
	}

	// extra parameters
	outputsize := values.Get(QueryOutputSize)
	if outputsize != "" {
		query.Set(QueryOutputSize, outputsize)
	}
	datatype := values.Get(QueryDataType)
	if datatype != "" {
		query.Set(QueryDataType, datatype)

	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// func limit(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if limiter.Allow() == false {
// 			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
// 			return
// 		}

// 		next.ServeHTTP(w, r)
// 	})
// }
