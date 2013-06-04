package types

type Login struct {
    Name string
    Password string
}

type LoginMsg struct {
	Login Login
	Resp chan LoginResp
}

func NewSubmit(n string, d []byte, k string) *Submit {
	return &Submit{Name: n, Data: d, Key: k}
}

type Submit struct {
    Name string
    Data []byte
    Key string
}

type LoginResp struct {
    Err string
    Key string
}

type PlayListItem struct {
    TrackName string
    Filename string
}

type PlaylistReq struct {
    ResultChan chan []string
}

type PlaylistAdd struct {
	TrackName string
	TrackFile string
}

const (
	permAdmin = 1 << iota
)

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
