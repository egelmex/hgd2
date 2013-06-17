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
	"strings"
	//	"time"
	"strconv"
	//	"os"
	"path"
)
import "github.com/jmhodges/levigo"

var db *levigo.DB

var initPassword string

var settings settings_t

type settings_t struct {
	port          uint
	dir           string
	maxUploadSize uint
}

func init() {
	const (
		defaultPort = 8080
		usagePort   = "port to connect to"

		usageDIR = "dir for HDG configuration files"

		usageInit       = "First time initialisation"
		defaultPassword = ""

		defaultMaxUploadSize = 100
		usageMaxUploadSize   = "Maximum upload size in MB."
	)

	defaultDIR := xdgbase.GetConfigHome() + "/hgd"

	flag.UintVar(&settings.port, "port", defaultPort, usagePort)
	flag.UintVar(&settings.port, "p", defaultPort, usagePort+" (shorthand)")

	flag.StringVar(&settings.dir, "dir", defaultDIR, usageDIR)
	flag.StringVar(&settings.dir, "d", defaultDIR, usageDIR+" (shorthand)")

	flag.StringVar(&initPassword, "init", defaultPassword, usageInit)

	flag.UintVar(&settings.maxUploadSize, "s", defaultMaxUploadSize, usageMaxUploadSize)

	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
}

func main() {
	done := make(chan int)
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		log.Println("Arguments were given parsing them first.")
		switch strings.ToLower(args[0]) {
		default:
			log.Println("unknown command")
		}
		log.Fatal("Quiting argument handled.")
	}

	log.Println("HGD STARTING")

	// Set up DB
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}

	// Set up user manager
	um := usermanager.NewUserManager(db)
	err = um.Initialise()
	log.Print("About to start UM")
	go um.Run()
	if err != nil && initPassword == "" {
		log.Fatal("No users set, if this is the first time you have run hgd use the init flag 'init=<password>'")
	} else if err != nil {
		resp := make(chan bool)
		log.Printf("Adding admin user")
		um.AddUser <- usermanager.AddUserMsg{usermanager.User{"admin", initPassword, 0xff}, resp}

		log.Printf("Waiting for admin user to be added.")
		r := <-resp
		if !r {
			log.Fatal("Failed to add admin user.")
		} else {
			log.Printf("Admin user added")
		}
	} else if initPassword != "" {
		//XXX: TODO handle this.
		log.Fatal("User DB was not blank, are you sure you want to add a new user")
	}

	// setup playlist manager.
	plm := playlist.NewPlaylistManager(db, settings.dir)
	go plm.Run()

	// Set up http
	http.HandleFunc("/login", mklogin(um.Login))
	http.HandleFunc("/playlist", mkplaylist(plm.RequestPlaylist))
	http.HandleFunc("/submit", mksubmit(plm.AddTrack, um.KeyCheck))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "HGD server, %q", html.EscapeString(r.URL.Path))
	})
	http.HandleFunc("/user/", mkuser(um))

	go func() {
		log.Println("Starting http server")
		err := http.ListenAndServe(":"+strconv.Itoa(int(settings.port)), nil)
		if err != nil {
			log.Println(err)
		}
		log.Println("listen and serve is done..")
		done <- 1
	}()

	log.Println("All done setting up.")
	<-done
}

func initDB() (*levigo.DB, error) {
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3 << 30))
	opts.SetCreateIfMissing(true)
	return levigo.Open(settings.dir+"/db", opts)

}

func mkuser(um usermanager.UserManager) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		userPart := path.Base(p)
		dirPart := path.Dir(p)
		log.Println(userPart)
		log.Println(dirPart)
		if r.Method == "POST" {
			if dirPart != "/user" {
				log.Println("unknown uri")
			}
			log.Printf("New addUser attempt")
			r.ParseForm()
			var jsonAddUser = r.FormValue("data")

			addUser := new(types.AddUser)

			var e = json.Unmarshal([]byte(jsonAddUser), addUser)
			if e != nil {

				w.WriteHeader(http.StatusBadRequest)
				return
			}
			permok, userok := checkPerms(um.KeyCheck, addUser.Key, usermanager.PermAddUser)
			if !userok {
				log.Println("Submit failed from unkown user key: \"", addUser.Key, "\"")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if !permok {
				log.Println("Submit failed userkey \"", addUser.Key, "\" has invalide perms")
				fmt.Fprintf(w, "{\"error\":\"Unknown User\"}")
				return
			}

			resp := make(chan bool)
			user := usermanager.User{
				Name:        userPart,
				Password:    addUser.Password,
				Permissions: 0,
			}

			if addUser.CanSubmit {
				user.Permissions |= usermanager.PermSubmit
			}

			if addUser.CanAddUsers {
				user.Permissions |= usermanager.PermAddUser
			}

			um.AddUser <- usermanager.AddUserMsg{user, resp}
			<-resp

		}
	}
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
			if e != nil {
				log.Printf("LoginFailed %v \"%v\"\n", e, r.FormValue("data"))
				return

			}
			log.Printf("Login Request from: '%v'", login.Name)
			resp := make(chan usermanager.LoginResp)
			out <- usermanager.LoginMsg{*login, resp}
			r := <-resp

			if r.Err != nil {
				fmt.Fprintf(w, "{\"Error\":\"%s\"}", r.Err)
				return
			}

			fmt.Fprintf(w, "{\"Key\":\"%s\"}", r.Key)
		} else {
			http.NotFound(w, r)
		}
	}
}
