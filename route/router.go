package route

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	staticRouter       = Router{make(map[string]*Role)}
	uriLabel           = make(map[string]string) //key(roleId):value(uri)
	paramLabel         = make(map[string]string) //key(roleId):value(param)
	fingerprintLibrary = make(map[string]string) // key(roleId):value(fingerprint)
	DefaultF           func() string
)

type Router struct {
	roles map[string]*Role
}

//添加路由规则
func AddRole(role *Role) bool {

	if len(role.UriRegular) == 0 {
		return false
	}

	pl := role.ParamLabel()
	if len(role.ParamRegular) != 0 {
		//有参数，校验规则是否相同
		for k, _ := range paramLabel {
			if strings.Contains(k, pl) || strings.Contains(pl, k) {
				return false
			}
		}
	} else {
		//没有参数规则，需校验uri是否有重叠
		for _, v := range uriLabel {
			if strings.HasPrefix(v, role.UriRegular) || strings.HasPrefix(role.UriRegular, v) {
				return false
			}
		}
	}

	fp := role.Fingerprint()
	for _, v := range fingerprintLibrary {
		if strings.Compare(v, fp) == 0 {
			return false
		}
	}
	staticRouter.roles[role.Id] = role
	uriLabel[role.Id] = role.UriRegular
	if len(role.ParamRegular) != 0 {
		paramLabel[role.Id] = pl
	}
	return true
}

func Distribute(r *http.Request) (*http.Request, bool) {
	r.ParseForm()
	url := ""
	for _, role := range staticRouter.roles {
		if hostname, def := role.Match(r); def {
			url = hostname
			fmt.Printf("URL:%s 命中:%s target:%s\n", r.RequestURI, role.Id, url)
			break
		}
	}

	if len(url) < 1 {
		//从所有的服务中默认获取一个
		url = "http://" + DefaultF() + r.URL.String()
		fmt.Printf("请求%s 匹配默认服务组:%s\n", r.RequestURI, url)
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
