package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
)

const addForm = `
<form method="POST" action="/add">
URL: <input type="text" name="url">
<input type="submit" value="Add">
</form>
`

var (
	listenAddr = flag.String("http", ":8080", "http listen address")
	dataFile   = flag.String("file", "store.json", "data store file name")
	hostname   = flag.String("host", "localhost:8080", "host name and port")

	masterAddr = flag.String("master", "", "RPC master address")
	rpcEnabled = flag.Bool("rpc", false, "enable RPC server")
)

var store Store

func main() {
	flag.Parse()

	if *masterAddr != "" {
		store = NewProxyStore(*masterAddr)
	} else {
		store = NewURLStore(*dataFile)
	}

	if *rpcEnabled { // the master is the rpc server
		rpc.RegisterName("Store", store)
		rpc.HandleHTTP()
	}

	http.HandleFunc("/", redirectHandler)
	http.HandleFunc("/add", addHandler)

	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	var url string

	if err := store.Get(&key, &url); err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, addForm)
	case "POST":
		url := r.FormValue("url")
		var key string

		if err := store.Put(&url, &key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "http://%s/%s", *hostname, key)
	}
}
