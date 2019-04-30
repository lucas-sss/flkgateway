package main

import (
	"context"
	"flkgateway/action"
	"flkgateway/route"
	"flkgateway/sniffing"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
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
	data, err := action.Do(req)
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
	req, err := http.NewRequest("GET", "http://192.168.20.187:8080/hello", nil)
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
	w.Write([]byte("config"))
}

func main() {

	//添加路由规则
	serverGroup1 := map[string]int{"192.168.20.187:8088": 1}
	serverGroup2 := map[string]int{"192.168.20.187:8089": 1}
	serverGroup3 := map[string]int{"192.168.20.187:8086": 3, "192.168.20.187:8087": 1}

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
	var b bool
	notice1 := make(chan map[string]bool, 1)
	role1 := (&route.Role{Id: "role1", UriRegular: "/hello/a", ParamMode: 0, ParamRegular: paramRegular1, ServerGroup: serverGroup1, Notice: notice1}).Init()
	b = route.AddRole(role1)
	//fmt.Println("role添加结果", b)

	notice2 := make(chan map[string]bool, 1)
	role2 := (&route.Role{Id: "role2", UriRegular: "/hello/a", ParamMode: 0, ParamRegular: paramRegular2, ServerGroup: serverGroup2, Notice: notice2}).Init()
	b = route.AddRole(role2)
	//fmt.Println("role添加结果", b)

	notice3 := make(chan map[string]bool, 1)
	role3 := (&route.Role{Id: "role3", UriRegular: "/hello/c", ServerGroup: serverGroup3, Notice: notice3}).Init()
	b = route.AddRole(role3)
	//fmt.Println("role添加结果", b)

	http.HandleFunc("/", routingHandler)
	http.HandleFunc("/config/", configHandler)
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	notice1 <- map[string]bool{"192.168.20.187:8088": false}
	//
	//	time.Sleep(10 * time.Second)
	//	notice1 <- map[string]bool{"192.168.20.187:8088": true}
	//	fmt.Println("-----------")
	//}()

	sniffing.HealthCheck()
	err := http.ListenAndServe(":"+strconv.Itoa(8080), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
