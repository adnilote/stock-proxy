package proxy

import (
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// OHLCV
type OHLCV struct {
	Timestamp int32
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Item struct is a document in mongoDB
type Item struct {
	ID     bson.ObjectId `json:"id" bson:"_id"`
	Ticker string        `json:"ticker" bson:"ticker"`
	Ohlcv  []bson.Binary `json:"ohlcv" bson:"ohlcv"`
}

// MongoDB instance of mongoDB, which can
// add and get documents to db
type MongoDB struct {
	db *mgo.Collection
}

// Add adds val by key to db
func (m *MongoDB) Add(key string, val []byte) error {
	key = strings.ToUpper(key)
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

// Get return record from db by key
func (m *MongoDB) Get(key string) (*Item, error) {
	item := Item{}
	err := m.db.Find(bson.M{"ticker": strings.ToUpper(key)}).One(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
