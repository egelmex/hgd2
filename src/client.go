package main

import ( "fmt"
	"log"
	"encoding/json"
	"net/http"
	"io/ioutil"
	"os"
	"lib/types"
	"flag"
	"net/url"
	"strings"
	"path")



var server string
var port string

var connectionString string

func init() {
	const (
		defaultServer = "localhost"
		usageServer   = "server to connect to"

        defaultPort = "8080"
        usagePort = "port to connect to"
	)
	flag.StringVar(&server, "server", defaultServer, usageServer)
	flag.StringVar(&server, "s", defaultServer, usageServer+" (shorthand)")

    flag.StringVar(&port, "port", defaultPort, usagePort)
	flag.StringVar(&port, "p", defaultPort, usagePort+" (shorthand)")
}

func main() {
    flag.Parse()
    args := flag.Args()
    connectionString = "http://" + server + ":" + port + "/"

    if len(args) == 0 {
        printPlayList()
    } else {
        switch strings.ToLower(args[0]) {
        case "playlist":
                printPlayList()
        case "submit":
                if len(args) == 2 {
                    submit(args[1:])
                } else {
                     fmt.Println("Submit <Filename>\n")

                }

        default:
                fmt.Println("Unknown command: ", args[0])
        }
    }


}

func submit(parms []string) {

        b, err := ioutil.ReadFile(parms[0])
        if err != nil { panic(err) }



        submit := types.NewSubmit(path.Base(parms[0]), b, "TODO")
        res, err := json.Marshal(submit)
        if err != nil {
            log.Fatal(err)
        }
        resp, err := http.PostForm(connectionString + "submit", url.Values{"data": {string(res)}})
        if err != nil {
            log.Fatal(err)
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            log.Fatal(err)
        }

        fmt.Printf(string(body))
}

func printPlayList() {
    resp, err := http.Get(connectionString + "playlist")
    if err != nil {
        log.Printf("error: %v\n", err)
       os.Exit(-1)
    }
    defer resp.Body.Close()

    r := new ([]string)

    decoder := json.NewDecoder(resp.Body)
    e := decoder.Decode(r)

    if e != nil {
	    log.Fatalf("Woops: %v", e)
    }

    fmt.Printf("Playlist:\n")
    for i, v := range(*r) {
	    fmt.Printf("%d: %v\n",i,v)
    }
}

