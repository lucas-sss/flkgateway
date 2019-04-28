package route

import (
	"io"
	"net/http"
	"strings"
)

type Router struct {
	roles map[string]*Role
}

var staticRouter = Router{make(map[string]*Role)}

func AddRole(role *Role) bool {
	staticRouter.roles[role.Id] = role
	return true
}

func Distribute(r *http.Request) (*http.Request, bool) {
	r.ParseForm()
	url := ""
	for _, role := range staticRouter.roles {
		if hostname, def := role.Match(r); def {
			url = hostname
			break
		}
	}

	if len(url) < 1 {
		url = "http://" + "192.168.20.187:8090" + r.URL.String()
	}

	var body io.Reader
	postParam := ""
	for k, v := range r.PostForm {
		if len(postParam) > 0 {
			postParam += "&"
		}
		postParam += k + "=" + v[0]
	}
	if len(postParam) > 0 {
		body = strings.NewReader(postParam)
	}

	//最后转发请求
	req, err := http.NewRequest(r.Method, url, body)
	if err != nil {
		// handle error
		panic("创建请求失败")
	}

	if r.Method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return req, false
}
