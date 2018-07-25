package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var GRAFANA_ROOT_URL string

func init() {
	GRAFANA_ROOT_URL = os.Getenv("GRAFANA_ROOT_URL")
	if len(GRAFANA_ROOT_URL) == 0 {
		GRAFANA_ROOT_URL = "http://grafana:3000"
	}
}

func main() {
	http.HandleFunc("/", proxy)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func proxy(w http.ResponseWriter, r *http.Request) {
	method := r.Method

	if method == "GET" {
		get(w, r)
	} else if method == "POST" {
		post(w, r)
	} else if method == "OPTIONS" {
		options(w, r)
	} else {
		fmt.Fprintf(w, "NOT Supported Method: %s", method)
	}
}

func options(w http.ResponseWriter, r *http.Request) {
	path := getPath(r)
	log.Printf("options: %s\n", path)
	req, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	execute(w, r, req)
}

func post(w http.ResponseWriter, r *http.Request) {
	path := getPath(r)
	log.Printf("post: %s\n", path)
	req, err := http.NewRequest("POST", path, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	execute(w, r, req)
}

func get(w http.ResponseWriter, r *http.Request) {
	path := getPath(r)
	log.Printf("get: %s\n", path)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	execute(w, r, req)
}

func getPath(r *http.Request) string {
	path := r.URL.Path
	query := ""
	for k, vs := range r.URL.Query() {
		for _, v := range vs {
			query += k + "=" + url.QueryEscape(v) + "&"
		}
	}
	if len(query) > 0 {
		// query = url.QueryEscape(query)
		// tmp := &url.URL{Path: query}
		// tmp := query
		path = path + "?" + query
		// path = path + "?" + query
	}

	return GRAFANA_ROOT_URL + path
}

func execute(w http.ResponseWriter, r *http.Request, newReq *http.Request) {
	for k, vs := range r.Header {
		for _, v := range vs {
			newReq.Header.Add(k, v)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(newReq)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), resp.StatusCode)
		return
	}

	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), resp.StatusCode)
		return
	}
	w.Write(body)
	// fmt.Printf("proxy done with body length %d, body: %s\n", len(body), string(body))
}
