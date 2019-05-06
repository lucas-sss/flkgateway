package gatewaymain

import (
	"encoding/json"
	"flag"
	"flkgateway/route"
	"flkgateway/sniffing"
	"flkgateway/transport"
	"flkgateway/util"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
)

var config *Config = &Config{}

var cfgPath *string = flag.String("c", "./conf.toml", "the file path of config for this service")
var port *int = flag.Int("p", 8080, "the port of service")

type Config struct {
	AllServer     map[string]string
	DefaultServer []string
	Roles         []route.Role
}

func routingHandler(w http.ResponseWriter, r *http.Request) {
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

func checkConfig() {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("check config failed, panic: %v", e)
			os.Exit(1)
		}
	}()

	flag.Parse()

	if util.IsExist(*cfgPath) {
		if _, err := toml.DecodeFile(*cfgPath, config); err != nil {
			fmt.Printf("parse config file error %v\n", err)
			os.Exit(1)
		}
		for _, v := range config.Roles {
			notice := make(chan map[string]bool, 1)
			v.Notice = notice
			role := v.Init()
			b := route.AddRole(role)
			if !b {
				b, _ := json.Marshal(role)
				fmt.Printf("add role failed:%s\n", string(b))
			}
		}
		return
	}
	fmt.Printf("the configuration file is no exist:%s\n", *cfgPath)
	os.Exit(1)
}

func appInit() {
	i, cw, gcd := -1, 0, 1
	tmpWeight, index := 0, 0
	availableServer := make(map[int]map[string]interface{})
	if len(config.DefaultServer) == 0 {
		fmt.Println("Warning default server group is empty, using all server group as default")
		defaultServer := make([]string, 0, 1)
		for k, _ := range config.AllServer {
			defaultServer = append(defaultServer, k)
		}
		config.DefaultServer = defaultServer
	}
	for _, server := range config.DefaultServer {
		availableServer[index] = map[string]interface{}{"hostname": server, "weight": 1}
		index++
		if tmpWeight == 0 {
			tmpWeight = 1
			continue
		}
		gcd = util.GCD(tmpWeight, 1)
	}

	route.DefaultF = route.AccessGroup(i, cw, gcd, availableServer)
	if len(availableServer) == 0 {
		panic("default availableServer generate error")
	}
	sniffing.HealthCheck()
}

func Main() {
	checkConfig()
	appInit()

	mux := http.NewServeMux()
	mux.HandleFunc("/", safeHandler(routingHandler))
	err := http.ListenAndServe(":"+strconv.Itoa(*port), mux)
	if err != nil {
		//fmt.Printf("add role failed:%s\n", string(b))
		log.Fatal("ListenAndServe: ", err.Error())
	}

}
