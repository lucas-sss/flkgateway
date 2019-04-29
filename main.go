package main

import (
	"flkgateway/action"
	"flkgateway/route"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
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
	serverGroup1 := map[string]int{"192.168.20.187:8088": 1}
	serverGroup2 := map[string]int{"192.168.20.187:8089": 1}

	processElement1 := route.ProcessElement{
		Value:     "0",
		Operation: "=",
		Attach:    []string{"HASH", "MOD"},
		S: map[string]interface{}{
			"MOD": 2,
		},
	}
	processElement2 := route.ProcessElement{
		Value:     "1",
		Operation: "=",
		Attach:    []string{"HASH", "MOD"},
		S: map[string]interface{}{
			"MOD": 2,
		},
	}

	paramRegular1 := map[string]route.ProcessElement{
		"a": processElement1,
	}
	paramRegular2 := map[string]route.ProcessElement{
		"a": processElement2,
	}

	notice1 := make(chan map[string]bool)
	role1 := (&route.Role{Id: "role1", ParamMode: 0, ParamRegular: paramRegular1, ServerGroup: serverGroup1, Notice: notice1}).Init()
	route.AddRole(role1)

	notice2 := make(chan map[string]bool)
	role2 := (&route.Role{Id: "role2", ParamMode: 0, ParamRegular: paramRegular2, ServerGroup: serverGroup2, Notice: notice2}).Init()
	route.AddRole(role2)

	http.HandleFunc("/", routingHandler)
	http.HandleFunc("/config/", configHandler)
	go func() {
		time.Sleep(10 * time.Second)
		notice1 <- map[string]bool{"192.168.20.187:8088": false}

		time.Sleep(10 * time.Second)
		notice1 <- map[string]bool{"192.168.20.187:8088": true}
		fmt.Println("-----------")
	}()
	err := http.ListenAndServe(":"+strconv.Itoa(8080), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
