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
    )

const (
    DIR = "/home/me92/.hgd"
)

func main() {

	playlistReq := make(chan types.PlaylistReq)
	playlistAdd := make(chan types.Submit)
	playlistNextSong := make(chan nextSong)

	http.HandleFunc("/login", login)
	http.HandleFunc("/playlist", mkplaylist(playlistReq))
	http.HandleFunc("/submit", mksubmit(playlistAdd))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "HGD server, %q", html.EscapeString(r.URL.Path))
	})

	go playlistManager(playlistReq, playlistAdd, playlistNextSong)
    go play(playlistNextSong)
	go log.Fatal(http.ListenAndServe(":8080", nil))
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

func playlistManager(in chan types.PlaylistReq, playlistAdd chan types.Submit, ns chan nextSong) {
	log.Println("Started playlist manger")
	playlist := []types.PlayListItem{{"track1", "boobies"}}
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

func mkplaylist(out chan types.PlaylistReq) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("user requested playlist")
		c := make(chan []string)
		out <- types.PlaylistReq{c}
		playlist := <-c
		res, _ := json.Marshal(playlist)
		//XXX: handel err
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
