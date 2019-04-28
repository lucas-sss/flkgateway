package route

import (
	"flkgateway/util"
	"net/http"
	"regexp"
	"strings"
)

type Role struct {
	Id string //规则id

	UriRegular string            //uri匹配正则
	ParamMode  int               //匹配模式，0:and  1:or
	Param      map[string]string //参数匹配

	ServerGroup map[string]int       //{"192.168.20.186:8088": 2},后端服务组,权重为1-10
	serverMark  map[string]bool      //服务组标记，是否可用 {"192.168.20.186:8088":true}
	f           func() string        //获取下一个匹配服务组
	notice      chan map[string]bool //接收通知服务是否可用
}

func (role *Role) Init() *Role {
	role.serverMark = make(map[string]bool)
	for k, _ := range role.ServerGroup {
		role.serverMark[k] = true
	}

	//已有server不可用
	/*badServer := []string{}
	for _, v := range badServer {
		if _, ok := role.serverMark[v]; ok {
			role.serverMark[v] = false
		}
	}*/

	go func(r *Role) {
		select {
		case change := <-role.notice:
			for k, v := range change {
				if _, ok := r.serverMark[k]; ok {
					r.serverMark[k] = v
				}
			}
			//重新生成serverGroup
			createServerGenerator(r)
		}
	}(role)
	createServerGenerator(role)
	return role
}

func createServerGenerator(role *Role) {
	var i, cw, gcd = -1, 0, 1 //i表示上一次选择的服务器, cw表示当前调度的权值, gcd当前所有权重的最大公约数
	if len(role.ServerGroup) < 1 {
		//
		return
	}
	tmpWeight, index := 0, 0
	availableServer := make(map[int]map[string]interface{})
	for k, v := range role.ServerGroup {
		availableServer[index] = map[string]interface{}{"hostname": k, "weight": v}
		index++
		if tmpWeight == 0 {
			tmpWeight = v
			continue
		}
		gcd = util.GCD(tmpWeight, v)
	}

	role.f = func() string {
		for {
			i = (i + 1) % len(availableServer)
			if i == 0 {
				cw = cw - gcd
				if cw <= 0 {
					cw = getMaxWeight(availableServer)
					if cw == 0 {
						return ""
					}
				}
			}

			if weight, _ := availableServer[i]["weight"].(int); weight >= cw {
				return availableServer[i]["hostname"].(string)
			}
		}
	}
}

func getMaxWeight(servers map[int]map[string]interface{}) int {
	max := 0
	for _, v := range servers {
		if weight, _ := v["weight"].(int); weight >= max {
			max = weight
		}
	}
	return max
}

func (role Role) Match(req *http.Request) (string, bool) {
	//score:匹配请求正确得分，total:role所要求的总分
	score, total := 0, 0
	routingKey := req.URL.Path

	if len(role.UriRegular) > 0 {
		// uri匹配
		total++
		if ok, _ := regexp.Match(role.UriRegular, []byte(routingKey)); ok {
			score++
		}
	}

	if len(role.Param) > 0 && len(req.Form) > 0 {
		// 参数匹配
		total++
		switch role.ParamMode {
		case 0:
			hits := 0
			for k, v := range role.Param {
				if strings.Compare(v, req.Form.Get(k)) == 0 {
					hits++
				}
			}
			if hits == len(role.Param) {
				score++
			}
			break
		case 1:
			for k, v := range role.Param {
				if strings.Compare(v, req.Form.Get(k)) == 0 {
					score++
					break
				}
			}
			break
		}
	}

	if score != total || score == 0 {
		//没有匹配成功
		return "", false
	}

	hostname := role.f()
	if len(hostname) > 0 {
		return "http://" + hostname + req.URL.String(), true
	}
	return "", false

}
