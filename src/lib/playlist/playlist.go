package playlist

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"github.com/jmhodges/levigo"
	"io/ioutil"
	"lib/types"
	"log"
	"os/exec"
	"time"
)

type NextSong struct {
	ResultChan chan string
}

type PlaylistReq struct {
	ResultChan chan []string
}

type PlaylistAdd struct {
	TrackName string
	TrackFile string
}

type PlaylistManager struct {
	RequestPlaylist chan PlaylistReq
	AddTrack        chan types.Submit
	nextSong        chan NextSong
	database        *levigo.DB
	dir             string
	playlist        []types.PlayListItem
}

func NewPlaylistManager(db *levigo.DB, filestoreDir string) PlaylistManager {
	plm := PlaylistManager{
		RequestPlaylist: make(chan PlaylistReq),
		AddTrack:        make(chan types.Submit),
		nextSong:        make(chan NextSong),
		database:        db,
		dir:             filestoreDir,
		playlist:        []types.PlayListItem{},
	}

	return plm
}

func (plm PlaylistManager) Run() {
	log.Println("Started playlist manger")

	go play(plm.nextSong)

	plm.playlist = getPlaylistFromDB(plm.database)
	running := true
	for running {
		select {
		case req := <-plm.RequestPlaylist:
			log.Println("hadeling request for playlist")
			playlistPublic := []string{}
			for _, el := range plm.playlist {
				playlistPublic = append(playlistPublic, el.TrackName)
			}

			req.ResultChan <- playlistPublic
			log.Println("playlist sent")
		case req := <-plm.AddTrack:
			log.Println("Adding track: ", req.Name)
			plm.playlist = playListAdd(plm.database, req, plm.playlist, plm.dir)
		case req := <-func() chan NextSong {
			if len(plm.playlist) > 0 {
				return plm.nextSong
			}
			return nil
		}():
			log.Println("getting next track's filename")
			nt := plm.playlist[0]
			plm.playlist = plm.playlist[1:]
			req.ResultChan <- nt.Filename

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

func playListAdd(db *levigo.DB, req types.Submit, playlist []types.PlayListItem, dir string) []types.PlayListItem {

	fo, err := ioutil.TempFile(dir+"/files", req.Name)
	if err != nil {
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
	if err != nil {
		log.Fatal(err)
	}
	wo.Close()
}

func play(requestSong chan NextSong) {
	for {
		log.Println("Asking for next song")
		c := make(chan string)
		requestSong <- NextSong{c}
		ns := <-c
		log.Println("nextsong is ", ns)
		cmd := exec.Command("mplayer", ns)
		err := cmd.Start()
		if err != nil {
			log.Println(err)
			/// XXX
		}

		// helps provent fork bombing, will only cause an pause if tracks are <1sec long or invalid.
		time.Sleep(time.Second)
		err = cmd.Wait()
		if err != nil {
			log.Println(err)
			/// XXX
		}
	}
}
