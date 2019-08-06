package proxy

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
type Item struct {
	Id     bson.ObjectId `json:"id" bson:"_id"`
	Ticker string        `json:"ticker" bson:"ticker"`
	Ohlcv  []bson.Binary `json:"ohlcv" bson:"ohlcv"`
}

type MongoDB struct {
	db *mgo.Collection
}

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
