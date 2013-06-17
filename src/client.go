package main

import (
	"bufio"
	"code.google.com/p/gopass"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/types"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

//	"github.com/GeertJohan/go.linenoise"
)

var server string
var port string
var username string
var password string

var connectionString string

func init() {
	const (
		defaultServer = "localhost"
		usageServer   = "server to connect to"

		defaultPort = "8080"
		usagePort   = "port to connect to"

		defaultUser = ""
		usageUser   = "User to authenticate as"

		defaultPassword = ""
		usagePassword   = "password to authenticate with"
	)
	flag.StringVar(&server, "server", defaultServer, usageServer)
	flag.StringVar(&server, "s", defaultServer, usageServer+" (shorthand)")

	flag.StringVar(&port, "port", defaultPort, usagePort)
	flag.StringVar(&port, "p", defaultPort, usagePort+" (shorthand)")

	flag.StringVar(&username, "username", defaultUser, usageUser)
	flag.StringVar(&username, "l", defaultUser, usageUser+" (shorthand)")

	flag.StringVar(&password, "password", defaultPassword, usagePassword)
	flag.StringVar(&password, "P", defaultPassword, usagePassword+" (shorthand)")

}

func main() {
	flag.Parse()
	args := flag.Args()
	connectionString = "http://" + server + ":" + port + "/"

	key := ""
	var err error
	if username != "" {
		key, err = login(username, password)
	}
	if err != nil {
		println("Failed to log in, some services may fail.")
	}

	if len(args) == 0 {
		printPlayList()
	} else {
		switch strings.ToLower(args[0]) {
		case "playlist":
			printPlayList()
		case "submit":
			if len(args) == 2 {
				submit(args[1:], key)
			} else {
				fmt.Println("Submit <Filename>\n")

			}
		case "adduser":
			if len(args) == 2 {
				addUser(args[1:], key)
			} else {
				fmt.Println("adduser <username>")
			}
		default:
			fmt.Println("Unknown command: ", args[0])
		}
	}

}

func login(username, password string) (string, error) {

	login := types.Login{username, password}
	res, err := json.Marshal(login)
	if err != nil {
		log.Fatal(err)
	}
	println(string(res))
	resp, err := http.PostForm(connectionString+"login", url.Values{"data": {string(res)}})
	if err != nil {
		log.Fatal(err)
	}

	r := new(types.LoginResp)

	decoder := json.NewDecoder(resp.Body)
	e := decoder.Decode(r)
	if e != nil {
		log.Fatal(e)
	}

	if r.Err != "" {
		return "", errors.New(r.Err)
	}

	return r.Key, nil

}

func addUser(parms []string, key string) {
	if strings.TrimSpace(key) == "" {
		log.Fatal("Not logged in, must be authenticated to add user.")
	}

	log.Println("Adding User ", parms[0])
	username := parms[0]
	password, e := gopass.GetPass("password: ")
	if e != nil {
		log.Fatal(e)
	}

	reader := bufio.NewReader(os.Stdin)
	print("Can user submit tracks [Y/n] ")
	var canSubmit bool = true
	{
		var tmpYN string
		tmpYN, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		} else {
			tmpYN = strings.TrimSpace(strings.ToLower(tmpYN))
			if tmpYN == "n" || tmpYN == "no" {
				canSubmit = false
			}
		}
	}

	print("Can user Addnew Users [y/N] ")
	var canAddUsers bool = false
	{
		var tmpYN string
		tmpYN, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		} else {
			tmpYN = strings.TrimSpace(strings.ToLower(tmpYN))
			if tmpYN == "y" || tmpYN == "yes" {
				canSubmit = true
			}
		}
	}

	user := types.AddUser{
		Password:    password,
		Key:         key,
		CanSubmit:   canSubmit,
		CanAddUsers: canAddUsers,
	}

	res, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.PostForm(connectionString+"user/"+username, url.Values{"data": {string(res)}})
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(string(body))
}

func submit(parms []string, key string) {

	b, err := ioutil.ReadFile(parms[0])
	if err != nil {
		panic(err)
	}

	submit := types.NewSubmit(path.Base(parms[0]), b, key)
	res, err := json.Marshal(submit)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.PostForm(connectionString+"submit", url.Values{"data": {string(res)}})
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

	r := new([]string)

	decoder := json.NewDecoder(resp.Body)
	e := decoder.Decode(r)

	if e != nil {
		log.Fatalf("Woops: %v", e)
	}

	fmt.Printf("Playlist:\n")
	for i, v := range *r {
		fmt.Printf("%d: %v\n", i, v)
	}
}
