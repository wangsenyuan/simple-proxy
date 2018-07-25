package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var GRAFANA_ROOT_URL string
var ADMIN_PASSWORD string
var ADMIN_USER string

func init() {
	GRAFANA_ROOT_URL = os.Getenv("GRAFANA_ROOT_URL")
	if len(GRAFANA_ROOT_URL) == 0 {
		GRAFANA_ROOT_URL = "http://grafana.monitoring:3000"
	}

	ADMIN_PASSWORD = os.Getenv("ADMIN_PASSWORD")
	if len(ADMIN_PASSWORD) == 0 {
		ADMIN_PASSWORD = "password"
	}

	ADMIN_USER = os.Getenv("ADMIN_USER")
}

func main() {
	http.HandleFunc("/", proxy)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkAuth(r *http.Request) bool {
	user := getCookie(r, "user")
	timestamp := getCookie(r, "timestamp")
	token := getCookie(r, "token")

	if user == "" || timestamp == "" || token == "" {
		return false
	}
	hash := GetMD5Hash(timestamp + "_" + user + "_" + ADMIN_PASSWORD)

	return hash == token
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCookie(r *http.Request, name string) string {
	cookie, _ := r.Cookie(name)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

const HTML = `
<html>
	<body>
		<form action="/auth" method="POST">
			<input type="text" name="user" placeholder="user name"/>
			<input type="password" name="password" placeholder="password"/>
			<button type="submit">Submit</button>
		</form>
	</body>
</html>

`

func redirectAuth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(HTML))
	w.WriteHeader(http.StatusOK)
}

func tryAuth(w http.ResponseWriter, r *http.Request) bool {
	method := r.Method
	if method != "POST" {
		return false
	}
	r.ParseForm()

	var user, password string

	for k, v := range r.Form {
		if k == "user" {
			user = v[0]
		} else if k == "password" {
			password = v[0]
		}
	}

	if user != ADMIN_USER || password != ADMIN_PASSWORD {
		return false
	}
	t := time.Now()
	timestamp := t.Format("20060102150405")
	token := GetMD5Hash(timestamp + "_" + user + "_" + ADMIN_PASSWORD)

	expire := time.Now().Add(20 * time.Minute) // Expires in 20 minutes
	cookie := http.Cookie{Name: "user", Value: user, Path: "/", Expires: expire, MaxAge: 86400}
	http.SetCookie(w, &cookie)
	cookie = http.Cookie{Name: "timestamp", Value: timestamp, Path: "/", Expires: expire, MaxAge: 86400}
	http.SetCookie(w, &cookie)
	cookie = http.Cookie{Name: "token", Value: token, Path: "/", Expires: expire, MaxAge: 86400}
	http.SetCookie(w, &cookie)

	// w.WriteHeader(302)
	redirectToHome(w, r)

	return true
}

func redirectToHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", 302)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := getPath(r)

	if method == "OPTIONS" && strings.Contains(path, "auth") {
		w.WriteHeader(http.StatusOK)
		return
	}

	authed := checkAuth(r)
	if !authed {

		if strings.Contains(path, "auth") {
			authed = tryAuth(w, r)
		}

		if !authed {
			redirectAuth(w, r)
		}
		return
	}

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
