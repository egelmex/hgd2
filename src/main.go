package main

import "net/http"
import "fmt"
import "html"
import "log"
import "encoding/json"
import ("lib/types"
    "io/ioutil"
    "bufio"
    "os/exec"
    "time"
    "encoding/gob"
    "bytes"
    "flag"
    )
import "github.com/jmhodges/levigo"

const (
    DIR = "/home/me92/.hgd"
)

var db *levigo.DB
var port string
var dir string


func init() {
	const (
		defaultPort = "8080"
		usagePort = "port to connect to"

		defaultDIR = DIR
		usageDIR = "dir for HDG configuration files"
	)
	flag.StringVar(&port, "port", defaultPort, usagePort)
	flag.StringVar(&port, "p", defaultPort, usagePort+" (shorthand)")

	flag.StringVar(&dir, "dir", defaultDIR, usageDIR)
	flag.StringVar(&dir, "d", defaultDIR, usageDIR +" (shorthand)")
}


func main() {

	log.Println("HGD STARTING")
	flag.Parse()

	var err error

	playlistReq := make(chan types.PlaylistReq)
	playlistAdd := make(chan types.Submit)
	playlistNextSong := make(chan nextSong)

	http.HandleFunc("/login", login)
	http.HandleFunc("/playlist", mkplaylist(playlistReq))
	http.HandleFunc("/submit", mksubmit(playlistAdd))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "HGD server, %q", html.EscapeString(r.URL.Path))
	})

	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3<<30))
	opts.SetCreateIfMissing(true)
	db, err = levigo.Open(DIR + "/db", opts)
	if err != nil {
		log.Fatal(err)
	}

	go playlistManager(db, playlistReq, playlistAdd, playlistNextSong)
	go play(playlistNextSong)
	go log.Fatal(http.ListenAndServe(":" + port, nil))

	time.Sleep(time.Second)
	log.Println("All done setting up.")
}


func mksubmit(out chan types.Submit) func(http.ResponseWriter, *http.Request) {
        return func(w http.ResponseWriter, r *http.Request) {
            if r.Method == "POST" {
                log.Printf("New submit attempt")
                r.ParseForm()
                var jsonLogin = r.FormValue("data")
                submit := new(types.Submit)
                var e = json.Unmarshal([]byte(jsonLogin), submit)
                if e == nil {
                    log.Printf("Submit Request from: %v", submit.Key)
                    out <- *submit
                } else {
                    log.Printf("SubmitFailed %v \"%v\"\n", e, r.FormValue("data"))
                }
            } else {
                http.NotFound(w, r)
            }
        }
}

func playlistManager(db *levigo.DB, in chan types.PlaylistReq, playlistAdd chan types.Submit, ns chan nextSong) {
	log.Println("Started playlist manger")

	playlist := getPlaylistFromDB(db)
	running := true
	for running {
        select {
        case req := <-in:
                log.Println("hadeling request for playlist")
                playlistPublic := []string{}
                for _, el := range playlist {
                    playlistPublic = append(playlistPublic, el.TrackName)
                }

                req.ResultChan <- playlistPublic
                log.Println("playlist sent")
        case req := <-playlistAdd:
                log.Println("Adding track: ", req.Name)
		playlist = playListAdd(req, playlist)
        case eq := <-func () chan nextSong {
                if len(playlist) > 0 {
                        return ns
                }
                return nil
        }():
                log.Println("getting next track's filename")
                nt := playlist[0]
		playlist = playlist[1:]
                eq.ResultChan <- nt.Filename

	    }
    }
    log.Println("playlistManager exited.")
}

func getPlaylistFromDB(db *levigo.DB) []types.PlayListItem {
	log.Println("Loading Playlist from file...")

	ro := levigo.NewReadOptions()

	data, err := db.Get(ro, []byte("playlist"))
	if err != nil {
		log.Fatal(err)
	}

	p := bytes.NewBuffer(data)

	dec := gob.NewDecoder(p)

	var playlist []types.PlayListItem
        //we must decode into a pointer, so we'll take the address of e 
        err = dec.Decode(&playlist)
        if err != nil {
                log.Print(err)
		playlist = []types.PlayListItem{}
        }

	log.Println("Loaded ", len(playlist), " items into playlist")
	log.Println(playlist)
	return playlist

}

func playListAdd(req types.Submit, playlist []types.PlayListItem) []types.PlayListItem {

                fo, err := ioutil.TempFile(DIR, req.Name)
                if (err != nil) {
                    log.Panic(err)
                    //XXX: handel error
		}
                defer func() {
			/// XXX: don't think this can run...
                    if err := fo.Close(); err != nil {
                        panic(err)
                    }
                }()

                w := bufio.NewWriter(fo)

                if _, err := w.Write(req.Data); err != nil {
                    panic(err)
                }

                playlist = append(playlist, types.PlayListItem{req.Name, fo.Name()})
		writePlaylistToDB(db, playlist)
		return playlist
}

func writePlaylistToDB(db *levigo.DB, playlist []types.PlayListItem) {
	wo := levigo.NewWriteOptions()
	m := new(bytes.Buffer)
	enc := gob.NewEncoder(m)
	enc.Encode(playlist)

	err := db.Put(wo, []byte("playlist"), m.Bytes())
	if (err != nil) {
		log.Fatal(err)
	}
	wo.Close()
}

func mkplaylist(out chan types.PlaylistReq) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("user requested playlist")
		c := make(chan []string)
		out <- types.PlaylistReq{c}
		playlist := <-c
		res, err := json.Marshal(playlist)
		if err != nil {
			log.Panic (err)
			//XXX: handel err
		}
		log.Printf("send user playist %v\n", string(res))
		fmt.Fprintf(w, "%s", res)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		log.Printf("New Login attempt")
		r.ParseForm()
		var jsonLogin = r.FormValue("data")
		login := new(types.Login)
		var e = json.Unmarshal([]byte(jsonLogin), login)
		if e == nil {
			log.Printf("Login Request from: %v", login.Name)
			fmt.Fprintf(w, "{\"Key\":\"TODO\"}")
		} else {
			log.Printf("LoginFailed %v \"%v\"\n", e, r.FormValue("data"))
		}
	} else {
		http.NotFound(w, r)
	}
}

type nextSong struct {
    ResultChan chan string
}


func play(requestSong chan nextSong) {
    for {
        log.Println("Asking for next song")
        c := make(chan string)
        requestSong <-nextSong{c}
        ns := <-c
        log.Println("nextsong is ", ns)
        cmd := exec.Command("mplayer", ns)
        err := cmd.Start()
        if (err != nil) {
            log.Println(err)
            /// XXX
        }
	err = cmd.Wait()
        if (err != nil) {
            log.Println(err)
            /// XXX
        }
	time.Sleep(time.Second)
    }
}
