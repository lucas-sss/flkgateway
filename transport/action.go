package transport

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var emptyData = RespData{}

type RespData struct {
	CL   int
	CT   string
	Data []byte
}

func Do(req *http.Request) (RespData, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		//
		return emptyData, err
	}
	defer resp.Body.Close()

	cl := resp.Header.Get("Content-Length")
	size, _ := strconv.Atoi(strings.Trim(cl, " "))
	ct := resp.Header.Get("Content-Type")
	body, err := ioutil.ReadAll(resp.Body)

	data := RespData{CL: size, CT: ct, Data: body}
	return data, nil

}
