// ixm - the Intelligent eXchange Monitor

package main

import (
	//"fmt"
	//"html"
	"encoding/json"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"log"
	"net/http"
	"runtime"
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

func main() {
	// set goroutine thread count to num CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	// connect to database
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatalln(err)
	}
	defer session.Close()

	// dynamic content
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		templateHandler(w, r, "index.html")
	})

	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		templateHandler(w, r, "info.html")
	})

	http.HandleFunc("/charts", func(w http.ResponseWriter, r *http.Request) {
		chartsHandler(w, r, session.Copy())
	})

	http.HandleFunc("/docs-api", func(w http.ResponseWriter, r *http.Request) {
		templateHandler(w, r, "docs-api.html")
	})

	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		templateHandler(w, r, "about.html")
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		templateHandler(w, r, "stats.html")
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHandler(w, r, session.Copy())
	})

	// static content
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./html/favicon.ico")
	})

	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./html/css"))))

	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("./html/fonts"))))

	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./html/js"))))

	// run http server
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Serve any template
func templateHandler(w http.ResponseWriter, r *http.Request, src string) {
	src = "template/" + src

	t, err := template.ParseFiles(src)
	if err != nil {
		log.Println(err)
	}

	t.Execute(w, nil)
}

// Serve charts
func chartsHandler(w http.ResponseWriter, r *http.Request, session *mgo.Session) {
	defer session.Close()

	// parse template
	t, err := template.ParseFiles("template/charts.html")
	if err != nil {
		log.Println(err)
	}

	// get arg
	market := r.URL.Query().Get("market")

	// get list of markets
	rawMarkets, err := getMarkets(session.Copy())
	if err != nil {
		log.Println(err)
	}

	// populate []markets
	var markets []string
	for _, entry := range rawMarkets {
		markets = append(markets, entry.Market)
	}

	// if arg
	if market != "" {
		// check if exists
		for _, entry := range rawMarkets {
			if market == entry.Market {
				// if exists, send data
				rawData, err := getTicker(session.Copy(), market)
				if err != nil {
					log.Println(err)
				}

				steps := [...]int{
					0, 1, 3, 6,
					12,	24, 48, 96,
				}

				iMap := rawData[0]["inferences"].(bson.M)
				rawMap := rawData[0]["input"].(bson.M)
				inferences := [...]float64{
					rawMap["value"].(float64),
					iMap["1"].(float64),
					iMap["3"].(float64),
					iMap["6"].(float64),
					iMap["12"].(float64),
					iMap["24"].(float64),
					iMap["48"].(float64),
					iMap["96"].(float64),
				}

				mMap := rawData[0]["metrics"].(bson.M)
				metrics := [...]float64{
					0.0,
					mMap["1"].(float64),
					mMap["3"].(float64),
					mMap["6"].(float64),
					mMap["12"].(float64),
					mMap["24"].(float64),
					mMap["48"].(float64),
					mMap["96"].(float64),
				}

				deltas := [...]float64{
					inferences[0] - inferences[0],
					inferences[1] - inferences[0],
					inferences[2] - inferences[0],
					inferences[3] - inferences[0],
					inferences[4] - inferences[0],
					inferences[5] - inferences[0],
					inferences[6] - inferences[0],
					inferences[7] - inferences[0],
				}

				model := ChartsModel{
					Market: market,
					Markets: markets,
					Timestamp: rawData[0]["timestamp"].(int),
					Steps: steps,
					Inferences: inferences,
					Metrics: metrics,
					Deltas: deltas,
				}

				t.Execute(w, model)
				return
			}
		}
	}

	// if no arg, no data
	model := ChartsModel{
		Market: "nil",
		Markets: markets,
		Timestamp: 0,
	}
	t.Execute(w, model)
}

// API Root, serves all available metrics
func apiHandler(w http.ResponseWriter, r *http.Request, session *mgo.Session) {
	defer session.Close()

	args := r.URL.Query()

	// set response type for switch
	response := "root"
	if _, ok := args["market"]; ok {
		response = "ticker"
		if _, ok := args["start"]; ok {
			response = "range"
		} else if _, ok := args["end"]; ok {
			response = "range"
		}
	}

	// switch on response to decide which dataset to send
	switch response {
	case "root":
		data, err := getMarkets(session.Copy())
		if err != nil {
			log.Println(err)
		}

		json.NewEncoder(w).Encode(data)
	case "ticker":
		market := r.URL.Query().Get("market")

		data, err := getTicker(session.Copy(), market)
		if err != nil {
			log.Println(err)
		}

		json.NewEncoder(w).Encode(data)
	case "range":
		market := r.URL.Query().Get("market")
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		data, err := getRange(session.Copy(), market, start, end)
		if err != nil {
			log.Println(err)
		}

		json.NewEncoder(w).Encode(data)
	}
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
