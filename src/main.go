package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"lib/playlist"
	"lib/types"
	"lib/usermanager"
	"lib/xdgbase"
	"log"
	"net/http"
	"time"
)
import "github.com/jmhodges/levigo"

var db *levigo.DB
var port string
var dir string

func init() {
	const (
		defaultPort = "8080"
		usagePort   = "port to connect to"

		usageDIR = "dir for HDG configuration files"
	)

	defaultDIR := xdgbase.GetConfigHome() + "/hgd"

	flag.StringVar(&port, "port", defaultPort, usagePort)
	flag.StringVar(&port, "p", defaultPort, usagePort+" (shorthand)")

	flag.StringVar(&dir, "dir", defaultDIR, usageDIR)
	flag.StringVar(&dir, "d", defaultDIR, usageDIR+" (shorthand)")
}

func main() {

	log.Println("HGD STARTING")
	flag.Parse()

	var err error

	playlistReq := make(chan playlist.PlaylistReq)
	playlistAdd := make(chan types.Submit)
	loginReq := make(chan usermanager.LoginMsg)
	keyCheck := make(chan usermanager.KeyCheckMsg)

	http.HandleFunc("/login", mklogin(loginReq))
	http.HandleFunc("/playlist", mkplaylist(playlistReq))
	http.HandleFunc("/submit", mksubmit(playlistAdd, keyCheck))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "HGD server, %q", html.EscapeString(r.URL.Path))
	})

	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3 << 30))
	opts.SetCreateIfMissing(true)
	db, err = levigo.Open(dir+"/db", opts)
	if err != nil {
		log.Fatal(err)
	}

	go playlist.PlaylistManager(db, playlistReq, playlistAdd, dir)
	go usermanager.UserManger(loginReq, keyCheck)
	go log.Fatal(http.ListenAndServe(":"+port, nil))

	time.Sleep(time.Second)
	log.Println("All done setting up.")
}

func mksubmit(out chan types.Submit, keyCheck chan usermanager.KeyCheckMsg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			log.Printf("New submit attempt")
			r.ParseForm()
			var jsonLogin = r.FormValue("data")
			submit := new(types.Submit)

			var e = json.Unmarshal([]byte(jsonLogin), submit)
			if e == nil {
				log.Printf("Submit Request from: %v", submit.Key)

				permok, userok := checkPerms(keyCheck, submit.Key, usermanager.PermSubmit)
				if !userok {
					log.Println("Submit failed from unkown user key: \"", submit.Key, "\"")
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if !permok {
					log.Println("Submit failed userkey \"", submit.Key, "\" has invalide perms")
					fmt.Fprintf(w, "{\"error\":\"Unknown User\"}")
					return
				}
				out <- *submit
			} else {
				log.Printf("SubmitFailed %v \"%v\"\n", e, r.FormValue("data"))
			}
		} else {
			http.NotFound(w, r)
		}
	}
}

func checkPerms(keyCheck chan usermanager.KeyCheckMsg, key string, perms int) (bool, bool) {
	resp := make(chan usermanager.KeyCheckResp)

	keyCheck <- usermanager.KeyCheckMsg{key, resp}
	r := <-resp

	if r.OK {
		if (r.Permissions & perms) == perms {
			return true, true
		} else {
			return false, true
		}
	} else {
		return false, false
	}

}

func mkplaylist(out chan playlist.PlaylistReq) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("user requested playlist")
		c := make(chan []string)
		out <- playlist.PlaylistReq{c}
		playlist := <-c
		res, err := json.Marshal(playlist)
		if err != nil {
			log.Panic(err)
			//XXX: handel err
		}
		log.Printf("send user playist %v\n", string(res))
		fmt.Fprintf(w, "%s", res)
	}
}

func mklogin(out chan usermanager.LoginMsg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			log.Printf("New Login attempt")
			r.ParseForm()
			var jsonLogin = r.FormValue("data")
			login := new(types.Login)
			var e = json.Unmarshal([]byte(jsonLogin), login)
			if e == nil {
				resp := make(chan usermanager.LoginResp)
				out <- usermanager.LoginMsg{*login, resp}
				r := <-resp

				log.Printf("Login Request from: %v", login.Name)
				fmt.Fprintf(w, "{\"Key\":\"%s\"}", r.Key)
			} else {
				log.Printf("LoginFailed %v \"%v\"\n", e, r.FormValue("data"))
			}
		} else {
			http.NotFound(w, r)
		}
	}
}
