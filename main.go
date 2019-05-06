package main

import (
	"flkgateway/route"
	"flkgateway/sniffing"
	"flkgateway/transport"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

func routingHandler(w http.ResponseWriter, r *http.Request) {
	//start := time.Now().UnixNano()
	//fmt.Println("start:", start)
	//1、拦截

	//2、根据uri与参数进行路由匹配
	req, def := route.Distribute(r)
	if !def {
		//没有匹配上,默认地址
	}

	//3、发送请求
	data, err := transport.Do(req)
	if err != nil {
		// handle error
	}

	w.Header().Set("Content-Type", data.CT)
	w.Write(data.Data)
	//stop := time.Now().UnixNano()
	//fmt.Println("stop:", stop)
	//fmt.Println("time consuming:", stop-start)
}
func configHandler(w http.ResponseWriter, r *http.Request) {
	reqURI := r.RequestURI
	fmt.Println("reqURI:", reqURI)
	rawQuery := r.URL.Path
	s := strings.Split(rawQuery, "/")

	path := strings.Replace(rawQuery, "/config/", "", -1)
	fmt.Println("path:", path)

	context := ""

	switch {
	case path == "":
		context = "index.html"
		//http.Redirect(w, r, "/config/list", 302)
		break
	case path == "list" || path == "list/":
		context = "list"
		break
	case path == "add" || path == "add/":
		context = "add"
		break
	case path == "update" || path == "update/":
		context = "update"
		break
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	fmt.Println("rawQuery:", s[2])

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(context))

	/*req, err := http.NewRequest("GET", "http://192.168.20.187:8080/hello", nil)
	if err != nil {
		log.Fatal(err)
	}
	context, _ := context.WithTimeout(context.Background(), 2*time.Second)
	req = req.WithContext(context)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		//TODO
		fmt.Print("error", err)
	}
	defer resp.Body.Close()

	fmt.Println("config...............")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("config"))*/
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				log.Printf("WARNING: panic in %v - %v", fn, e)
				log.Println(string(debug.Stack()))
			}
		}()
		fn(writer, request)
	}
}

type Config struct {
	AllServer map[string]string
	Roles     []route.Role
}

func main() {

	var config Config
	if _, err := toml.DecodeFile("./conf.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range config.Roles {
		role := v.Init()
		notice := make(chan map[string]bool, 1)
		role.Notice = notice
		b := route.AddRole(role)
		fmt.Println(v.Id, "添加结果: ", b)
	}

	http.HandleFunc("/", safeHandler(routingHandler))
	http.HandleFunc("/config/", safeHandler(configHandler))
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	notice1 <- map[string]bool{"192.168.20.187:8088": false}
	//
	//	time.Sleep(10 * time.Second)
	//	notice1 <- map[string]bool{"192.168.20.187:8088": true}
	//	fmt.Println("-----------")
	//}()

	sniffing.HealthCheck()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/config/", safeHandler(configHandler))
		err := http.ListenAndServe(":"+strconv.Itoa(8888), mux)
		if err != nil {
			log.Fatal("ListenAndServe: ", err.Error())
		}
	}()

	//err := http.ListenAndServe(":"+strconv.Itoa(8080), nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/", safeHandler(routingHandler))
	err := http.ListenAndServe(":"+strconv.Itoa(8080), mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
