package types

type Login struct {
    Name string
    Password string
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
