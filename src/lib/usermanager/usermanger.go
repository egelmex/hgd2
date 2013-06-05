package usermanager

import (
	"lib/types"
	"log"
	"crypto/rand"
	"encoding/hex"
)


const (
	PermSubmit = 1 << iota
)

type LoginMsg struct {
	Login types.Login
	Resp chan LoginResp
}

type User struct {
	Name string
	Password string
	Permissions int
}

type KeyCheckMsg struct {
	Key string
	Resp chan KeyCheckResp
}

type KeyCheckResp struct {
	Permissions int
	OK bool
}

type LoginResp struct {
    Err string
    Key string
}

func UserManger(login chan LoginMsg, checkkey chan KeyCheckMsg) {
	users := map [string] User{}
	keys := map [string] User{}
	addUser ("mex", "boobies", users, PermSubmit)
	for {
		select {
		case req := <-login:
			log.Println("Checking login of: ", req.Login.Name)

			user, ok := users[req.Login.Name]
			if !ok {
				req.Resp <- LoginResp{"Unknown username or password.",""}
			} else {
				if user.Password == req.Login.Password {
					///XXX: needs togenerate key
					key, _ := generateUUID()
					keys[key] = user
					req.Resp <- LoginResp{"", key}
				} else {
					req.Resp <- LoginResp{"Unknown username or password.",""}
				}
			}
		case req := <-checkkey:
			key, ok := keys[req.Key]
			req.Resp <- KeyCheckResp{key.Permissions, ok}
		}
	}
}

func addUser(username, password string, users map [string] User, permissions int){
	log.Print("Adding user ", username)
	users[username] = User{username, password, permissions}
}


func generateUUID() (string, bool) {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
	    return "", false
	}

	u[8] = (u[8] | 0x80) & 0xBF // what's the purpose ?
	u[6] = (u[6] | 0x40) & 0x4F // what's the purpose ?
	return hex.EncodeToString(u), true
}
