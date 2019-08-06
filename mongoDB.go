package main

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type OHLCV struct {
	Timestamp int32
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

type Request []OHLCV

type Item struct {
	Id     bson.ObjectId `json:"id" bson:"_id"`
	Ticker string        `json:"ticker" bson:"ticker"`
	Ohlcv  []bson.Binary `json:"ohlcv" bson:"ohlcv"`
	// Ohlcv []Request
}
type MetaData struct {
	Info       string `json:"1. Information" bson: "info"`
	Symbol     string `json:"2. Symbol" bson: "symbol"`
	Refreshed  string `json:"3. Last Refreshed" bson: "refreshed"`
	Intervel   string `json:"4. Interval" bson: "interval"`
	OutputSize string `json:"5. Output Size" bson: "size"`
	TimeZone   string `json:"6. Time Zone" bson: "time"`
}

type MongoDB struct {
	db *mgo.Collection
}

// func (m *MongoDB) AddVal(key string, rawVal []byte, format string) error {
// 	switch format {
// 	case "json":
// 		var item map[string]interface{}
// 		err := json.Unmarshal(rawVal, &item)
// 		if err != nil {

// 		}
// 		if _, ok := item["Meta Data"]; ok {
// 			h := md5.New()
// 			h.Write(item["Meta Data"].([]byte))
// 			bs := h.Sum(nil)
// 		}
// 	case "csv":
// 		ts, err := parseTimeSeriesData(bytes.NewReader(rawVal))

// 	default:
// 		return fmt.Errorf("Unknown format")
// 	}
// 	return nil
// }
func (m *MongoDB) Add(key string, val []byte) error {
	record := bson.M{
		"ticker": key,
	}
	err := m.db.Find(record).One(&Item{})

	if err != nil {
		// create new record in db
		err := m.db.Insert(bson.M{
			"ticker": key,
			"ohlcv": []bson.Binary{
				bson.Binary{Kind: 0x00, Data: val},
			},
		})
		if err != nil {
			return err
		}
		return nil
	}

	// Add ohlcv to db
	change := bson.M{"$push": bson.M{"ohlcv": bson.Binary{Kind: 0x00, Data: val}}}
	err = m.db.Update(record, change)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongoDB) Get(key string) (*Item, error) {
	// Get record from db
	item := Item{}
	err := m.db.Find(bson.M{"ticker": key}).One(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
