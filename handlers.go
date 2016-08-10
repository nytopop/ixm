// ixm - the Intelligent eXchange Monitor

// handlers.go

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
)

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
