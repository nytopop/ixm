// ixm - the Intelligent eXchange Monitor

// ixm.go

package main

import (
	"log"
	"net/http"
	"runtime"

	"gopkg.in/mgo.v2"
)

func main() {
	// set goroutine thread count to num CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	// connect to database
	session, err := mgo.Dial("mongodb://172.18.2.21")
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
