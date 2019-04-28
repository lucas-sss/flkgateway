package main

import (
	"flkgateway/action"
	"flkgateway/route"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func routingHandler(w http.ResponseWriter, r *http.Request) {
	//1、拦截

	//2、根据uri与参数进行路由匹配
	req, def := route.Distribute(r)
	if !def {
		//没有匹配上,默认地址
	}

	//3、发送请求
	data, err := action.Do(req)
	if err != nil {
		// handle error
	}

	w.Header().Set("Content-Type", data.CT)
	w.Write(data.Data)
}
func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("config...............")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("config"))
}

func main() {

	//添加路由规则
	serverGroup := map[string]int{"192.168.20.187:8088": 3, "192.168.20.187:8089": 6}

	role := (&route.Role{Id: "role1", ParamMode: 0, Param: map[string]string{"a": "1"}, ServerGroup: serverGroup}).Init()
	route.AddRole(role)

	http.HandleFunc("/", routingHandler)
	http.HandleFunc("/config/", configHandler)
	err := http.ListenAndServe(":"+strconv.Itoa(8080), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
