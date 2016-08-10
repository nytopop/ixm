// ixm - the Intelligent eXchange Monitor

// api.go

package main

import (
	//"fmt"
	//"html"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strconv"
)

// API Market response struct
type ApiMarket struct {
	Market  string `json:"market"`
	Records int    `json:"records"`
	First   int    `json:"first"`
	Last    int    `json:"last"`
}

// Charts model
type ChartsModel struct {
	Market		string
	Markets		[]string
	Timestamp	int
	Steps		[8]int
	Inferences	[8]float64
	Metrics		[8]float64
	Deltas		[8]float64
}

// Return all available markets
func getMarkets(session *mgo.Session) ([]ApiMarket, error) {
	defer session.Close()
	db := session.DB("ixm")

	data := []ApiMarket{}

	names, err := db.CollectionNames()
	if err != nil {
		return []ApiMarket{}, err
	}

	for _, name := range names {
		// skip index collection
		if name != "system.indexes" {
			// set col
			c := db.C(name)

			// set record count
			count, err := c.Count()
			if err != nil {
				return []ApiMarket{}, err
			}

			// find first/last timestamp
			rawFirst := bson.M{}
			err = c.Find(nil).Sort("timestamp").Limit(1).Select(bson.M{"timestamp": 1}).One(&rawFirst)
			if err != nil {
				return []ApiMarket{}, err
			}

			rawLast := bson.M{}
			err = c.Find(nil).Sort("-timestamp").Limit(1).Select(bson.M{"timestamp": 1}).One(&rawLast)
			if err != nil {
				return []ApiMarket{}, err
			}

			first := rawFirst["timestamp"].(int)
			last := rawLast["timestamp"].(int)

			// construct response
			data = append(data, ApiMarket{
				Market:  name,
				Records: count,
				First:   first,
				Last:    last,
			})
		}
	}

	// ret no err
	return data, nil
}

// return latest datapoint for market
func getTicker(session *mgo.Session, market string) ([]bson.M, error) {
	defer session.Close()
	db := session.DB("ixm")

	// let's use a slice of bson.M instead of ApiDatapoint
	data := []bson.M{}

	// check if the collection exists first
	names, err := db.CollectionNames()
	if err != nil {
		return []bson.M{}, err
	}

	// if name matches market, create c
	for _, name := range names {
		if name != "system.indexes" {
			if market == name {
				// access latest by timestamp, add to data
				c := db.C(name)

				err = c.Find(nil).Sort("-timestamp").Limit(1).Iter().All(&data)
				if err != nil {
					return []bson.M{}, err
				}
			}
		}
	}

	// ret no err
	return data, nil
}

// make this the range function, get rid of fromto
// if either start or end is nil,
func getRange(session *mgo.Session, market string, rawStart string, rawEnd string) ([]bson.M, error) {
	defer session.Close()
	db := session.DB("ixm")

	start, err := strconv.ParseInt(rawStart, 10, 64)
	if err != nil {
		return []bson.M{}, err
	}

	end, err := strconv.ParseInt(rawEnd, 10, 64)
	if err != nil {
		return []bson.M{}, err
	}

	if end == 0 {
		end = 9999999999
	}

	data := []bson.M{}

	// check if collection name exists
	names, err := db.CollectionNames()
	if err != nil {
		return []bson.M{}, err
	}

	// if name matches market, create c
	for _, name := range names {
		if name != "system.indexes" {
			if market == name {
				// using name from available to sanitize
				c := db.C(name)

				// get all docs where timestamp => start
				// no length limit, allow whole db dump
				query := bson.M{
					"timestamp": bson.M{
						"$gte": start,
						"$lte": end,
					},
				}

				err = c.Find(query).Iter().All(&data)
				if err != nil {
					return []bson.M{}, err
				}
			}
		}
	}

	// ret no err
	return data, nil
}
