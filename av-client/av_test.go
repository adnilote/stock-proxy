package av

import (
	"net/url"
	"testing"
)

const (
	proxyURL string = "http://127.0.0.1:8082"
)

type TestCase struct {
	params url.Values
	result string
}

func TestURL(t *testing.T) {
	av := NewAvClient("7Z29L509PNF9IE24")

	cases := []TestCase{
		TestCase{
			params: url.Values{
				QueryFunction: {"TIME_SERIES_DAILY_ADJUSTED"},
				QuerySymbol:   {"amzn"},
			},
			result: "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_DAILY_ADJUSTED&symbol=amzn",
		},
		TestCase{
			params: url.Values{
				QueryFunction: {"TIME_SERIES_DAILY_ADJUSTED"},
				QuerySymbol:   {"amzn"},
				QueryDataType: {"csv"},
			},
			result: "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&datatype=csv&function=TIME_SERIES_DAILY_ADJUSTED&symbol=amzn",
		},
		TestCase{
			params: url.Values{
				QueryFunction: {"TIME_SERIES_INTRADAY"},
				QueryInterval: {"5min"},
				QuerySymbol:   {"amzn"},
			},
			result: "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_INTRADAY&interval=5min&symbol=amzn",
		},
		TestCase{
			params: url.Values{
				QueryFunction:   {"TIME_SERIES_INTRADAY"},
				QueryInterval:   {"1min"},
				QuerySymbol:     {"msft"},
				QueryOutputSize: {"full"},
			},
			result: "https://www.alphavantage.co/query?apikey=7Z29L509PNF9IE24&function=TIME_SERIES_INTRADAY&interval=1min&outputsize=full&symbol=msft",
		},
		TestCase{
			params: url.Values{
				QueryFunction: {"TIME_SERIES_INTRADAY"},
				QuerySymbol:   {"amzn"},
			},
			result: "",
		},
		TestCase{
			params: url.Values{
				QueryFunction: {"TIME_SERIES_DAILY_ADJUSTED"},
			},
			result: "",
		},
	}

	for caseNum, cs := range cases {
		params := url.Values{}
		for key, val := range cs.params {
			params.Set(key, val[0])
		}
		u, _ := url.ParseRequestURI(proxyURL)
		u.Path = "/sync/"
		u.RawQuery = params.Encode()

		res, _ := av.URL(cs.params)
		if res != cs.result {
			t.Errorf("[%d] Got %s, expected %s", caseNum, res, cs.result)
		}
	}

}
