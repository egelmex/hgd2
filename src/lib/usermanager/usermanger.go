package usermanager

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"github.com/jmhodges/levigo"
	"lib/types"
	"log"
)

const (
	PermSubmit  = 1 << iota
	PermAddUser = 1 << iota
)

type AddUserMsg struct {
	User User
	Resp chan bool
}

type LoginMsg struct {
	Login types.Login
	Resp  chan LoginResp
}

type LoginResp struct {
	Err error
	Key string
}

type User struct {
	Name        string
	Password    string
	Permissions int
}

type KeyCheckMsg struct {
	Key  string
	Resp chan KeyCheckResp
}

type KeyCheckResp struct {
	Permissions int
	OK          bool
}

type UserManager struct {
	AddUser     chan AddUserMsg
	Login       chan LoginMsg
	KeyCheck    chan KeyCheckMsg
	initialised bool
	users       map[string]User
	keys        map[string]User
	database    *levigo.DB
}

func NewUserManager(db *levigo.DB) UserManager {
	login := make(chan LoginMsg)
	checkkey := make(chan KeyCheckMsg)
	adduser := make(chan AddUserMsg)

	m := UserManager{
		AddUser:     adduser,
		Login:       login,
		KeyCheck:    checkkey,
		initialised: false,
		users:       map[string]User{},
		keys:        map[string]User{},
		database:    db,
	}

	return m
}

func (um *UserManager) Initialise() error {
	log.Println("Initialising UserManager.")
	um.users = loadUsers(um.database)
	if len(um.users) == 0 {
		log.Printf("No user loaded")
		return errors.New("No users loaded")
	}
	um.initialised = true
	log.Printf("%+v", um.users)
	return nil
}

func (um *UserManager) Run() error {
	log.Printf("Starting user manager")
	log.Printf("%+v", um.users)

	for {
		select {
		case req := <-um.Login:
			handleLogin(req, um)
		case req := <-um.KeyCheck:
			key, ok := um.keys[req.Key]
			req.Resp <- KeyCheckResp{key.Permissions, ok}
		case req := <-um.AddUser:
			log.Println("Adding User.")
			/// XXX: Handel if user already exsists.
			log.Printf("%+v", req.User)
			um.users[req.User.Name] = req.User
			req.Resp <- true
			writeUsersToDB(um.database, um.users)
			log.Println("User added..")
		}
	}
}

func handleLogin(req LoginMsg, um *UserManager) {

	var (
		key string = ""
		err error  = nil
	)

	log.Printf("Checking login of: '%v' password: '%v'\n", req.Login.Name, req.Login.Password)

	user, ok := um.users[req.Login.Name]
	if !ok {
		log.Printf("Login failed, unknown username %v", req.Login.Name)
		err = errors.New("Unknown username or password.")
	} else {
		if user.Password == req.Login.Password {
			key, _ = generateUUID()
			um.keys[key] = user
		} else {
			log.Println("Login failed, wrong passoword for", req.Login.Name)
			err = errors.New("Unknown username or password.")
		}
	}
	req.Resp <- LoginResp{err, key}
	log.Println("Done checking login.")
}

func generateUUID() (string, bool) {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return "", false
	}

	u[8] = (u[8] | 0x80) & 0xBF
	u[6] = (u[6] | 0x40) & 0x4F
	return hex.EncodeToString(u), true
}

func loadUsers(db *levigo.DB) map[string]User {
	log.Println("Loading Users from file...")

	ro := levigo.NewReadOptions()

	data, err := db.Get(ro, []byte("Users"))
	if err != nil {
		log.Fatal(err)
	}

	p := bytes.NewBuffer(data)

	dec := gob.NewDecoder(p)

	var users map[string]User
	//we must decode into a pointer, so we'll take the address of e
	err = dec.Decode(&users)
	if err != nil {
		log.Print(err)
		users = map[string]User{}
	}

	log.Println("Loaded ", len(users), " users")
	log.Printf("%+v", users)
	return users

}

func writeUsersToDB(db *levigo.DB, users map[string]User) {
	wo := levigo.NewWriteOptions()
	m := new(bytes.Buffer)
	enc := gob.NewEncoder(m)
	enc.Encode(users)

	err := db.Put(wo, []byte("Users"), m.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	wo.Close()
}
