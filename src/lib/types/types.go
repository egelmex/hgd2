package types

type Login struct {
    Name string
    Password string
}

type LoginResp struct {
    Err string
    Key string
}

func NewSubmit(n string, d []byte, k string) *Submit {
	return &Submit{Name: n, Data: d, Key: k}
}

type Submit struct {
    Name string
    Data []byte
    Key string
}

type PlayListItem struct {
    TrackName string
    Filename string
}




