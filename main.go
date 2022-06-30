// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
    "fmt"
	"net/http"
    "encoding/json"

    "github.com/gorilla/sessions"

)

var hubs map[string]*Hub = make(map[string]*Hub)

var addr = flag.String("addr", ":8080", "http service address")

var clientNo = 0

func corsHandle(w http.ResponseWriter, r *http.Request){
    w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Credentials", "true")
}
// Check if user is authenticated
func validateSession(w http.ResponseWriter, r *http.Request) bool{

    session, _ := store.Get(r, "cookie-name")

    if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
        fmt.Printf("Validation failed %v\n", session )
        http.Error(w, "Forbidden", http.StatusForbidden)
        return false
    }
    fmt.Printf("Validation succeeded %v\n", session )
    return true
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	corsHandle(w,r)

    log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

var (
    // key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
    key = []byte("super-secret-key")
    store = sessions.NewCookieStore(key)
)

func secret(w http.ResponseWriter, r *http.Request) {
	corsHandle(w,r)

    if valid := validateSession(w,r); !valid {
        return
    }

    fmt.Println("Secret called")


    // Print secret message
    fmt.Fprintln(w, "The cake is a lie!")
}

func login(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Login called")
	corsHandle(w,r)

    session, _ := store.Get(r, "cookie-name")

    // Authentication goes here
    // ...

    // Set user as authenticated

    clientNo++
    userid:= clientNo
    username:= fmt.Sprintf("User %d",clientNo)

    session.Values["userid"]= userid
    session.Values["username"] = username

    session.Values["authenticated"] = true
    session.Save(r, w)

    msgJSON, _ := json.Marshal(
        map[string]interface{}{
            "id": userid,
            "username": username,
        })
    fmt.Fprint(w, string(msgJSON))
}

func logout(w http.ResponseWriter, r *http.Request) {
	corsHandle(w,r)

    session, _ := store.Get(r, "cookie-name")

    // Revoke users authentication
    session.Values["authenticated"] = false
    session.Save(r, w)
}

func auth(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Auth called")
	corsHandle(w,r)

    if valid := validateSession(w,r); !valid {
        return
    }
    session, _ := store.Get(r, "cookie-name")

    msgJSON, _ := json.Marshal(
        map[string]interface{}{
            "id": session.Values["userid"].(int),
            "username": session.Values["username"].(string),
        })

    fmt.Println(string(msgJSON))
    fmt.Fprint(w, string(msgJSON))
}

func websocketEndpoint(w http.ResponseWriter, r *http.Request){
    corsHandle(w,r)

    if valid := validateSession(w,r); !valid {
        return
    }

    serveWs(w, r)
}


func main() {
	flag.Parse()
    hub := newHub()
	go hub.run()
    hubs["main"] = hub

    http.HandleFunc("/secret", secret)
    http.HandleFunc("/login", login)
    http.HandleFunc("/logout", logout)
    http.HandleFunc("/auth", auth)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", websocketEndpoint)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
