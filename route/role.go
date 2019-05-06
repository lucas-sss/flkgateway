package route

import (
	"bytes"
	"flkgateway/util"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	LT    = "<"
	GT    = ">"
	EQUAL = "="
)

const (
	MOD  = "MOD"
	HASH = "HASH"
)

type ProcessElement struct {
	Value     string
	Operation string
	Attach    []string
	S         map[string]interface{}
}

//路由参数的签名，保证参数拦截不重复
//针对附加操作数S 的value只判定int、string和[]string类型，其余默认值为"1"
func (pe ProcessElement) Signature() string {
	var buffer bytes.Buffer
	buffer.WriteString(pe.Value)
	buffer.WriteString(pe.Operation)
	for _, v := range pe.Attach {
		buffer.WriteString(v)
		var s string
		if any, ok := pe.S[v]; ok {
			//
			switch v := any.(type) {
			case int:
				s = strconv.Itoa(v)
			case string:
				s = v
			case []string:
				for _, _v := range v {
					s += _v
				}
			default:
				s = "1"
			}
			buffer.WriteString(s)
		}
	}
	return buffer.String()
}

type Role struct {
	Id string //规则id

	UriRegular   string                    //uri匹配（前缀匹配），不能为空
	ParamMode    int                       //匹配模式，0:and  1:or
	ParamRegular map[string]ProcessElement //参数匹配

	ServerGroup map[string]int       //{"192.168.20.186:8088": 2},后端服务组,权重为1-10
	Notice      chan map[string]bool //接收通知服务是否可用
	serverMark  map[string]bool      //服务组标记，是否可用 {"192.168.20.186:8088":true}
	f           func() string        //获取下一个匹配服务组
}

func (role Role) UriLabel() string {
	if len(role.UriRegular) < 1 {
		return ""
	}
	return role.UriRegular
}

func (role Role) ParamLabel() string {
	if len(role.ParamRegular) < 1 {
		return ""
	}
	var buffer bytes.Buffer
	arr := make([]string, len(role.ParamRegular))
	i := 0
	for k, _ := range role.ParamRegular {
		arr[i] = k
	}
	sort.Strings(arr)
	for _, v := range arr {
		buffer.WriteString(v)
		buffer.WriteString(role.ParamRegular[v].Signature())
	}
	return buffer.String()
}

// 计算路由规则的指纹，去重
func (role Role) Fingerprint() string {
	var buffer bytes.Buffer
	buffer.WriteString(role.UriRegular)
	buffer.WriteString(strconv.Itoa(role.ParamMode))
	buffer.WriteString(role.ParamLabel())

	fingerprint := buffer.String()
	return util.MD5(fingerprint)
}

func (role Role) Init() *Role {
	if len(role.ServerGroup) == 0 {
		panic("Role init error: ServerGroup is nil")
	}
	rPt := new(Role)
	*rPt = role
	rPt.serverMark = make(map[string]bool)
	for k, _ := range role.ServerGroup {
		rPt.serverMark[k] = true
	}
	//已有server不可用
	/*badServer := []string{}
	for _, v := range badServer {
		if _, ok := role.serverMark[v]; ok {
			role.serverMark[v] = false
		}
	}*/

	go func(r *Role) {

		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("Runtime error caught: %v \n", rec)
			}
		}()
		if r.Notice == nil {
			return
		}
		for {
			select {
			case change := <-role.Notice:
				fmt.Println("receive:", change)
				for k, v := range change {
					if _, ok := r.serverMark[k]; ok {
						r.serverMark[k] = v
					}
				}
				//重新生成serverGroup
				createServerGenerator(r)
			}
		}
	}(rPt)
	createServerGenerator(rPt)
	return rPt
}

func AccessGroup(i, cw, gcd int, sg map[int]map[string]interface{}) func() string {

	return func() string {
		for {
			i = (i + 1) % len(sg)
			if i == 0 {
				cw = cw - gcd
				if cw <= 0 {
					cw = getMaxWeight(sg)
					if cw == 0 {
						return ""
					}
				}
			}
			if weight, _ := sg[i]["weight"].(int); weight >= cw {
				return sg[i]["hostname"].(string)
			}
		}
	}
}

func createServerGenerator(role *Role) {
	if len(role.ServerGroup) < 1 {
		panic(role.Id + " no server group")
	}
	i, cw, gcd := -1, 0, 1 //i表示上一次选择的服务器, cw表示当前调度的权值, gcd当前所有权重的最大公约数
	tmpWeight, index := 0, 0
	availableServer := make(map[int]map[string]interface{})
	for k, v := range role.ServerGroup {
		if !role.serverMark[k] {
			continue
		}
		availableServer[index] = map[string]interface{}{"hostname": k, "weight": v}
		index++
		if tmpWeight == 0 {
			tmpWeight = v
			continue
		}
		gcd = util.GCD(tmpWeight, v)
	}
	if len(availableServer) == 0 {
		panic(role.Id + " no available server")
	}
	role.f = AccessGroup(i, cw, gcd, availableServer)
	/*role.f = func() string {
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
	}*/
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
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("Runtime error caught: %v \n", rec)
		}
	}()

	//score:匹配请求正确得分，total:role所要求的总分
	score, total := 0, 0
	routingKey := req.URL.Path

	if len(role.UriRegular) > 0 {
		// uri匹配
		total++
		if strings.HasPrefix(routingKey, role.UriRegular) {
			score++
		}
		/*if ok, _ := regexp.Match(role.UriRegular, []byte(routingKey)); ok {
			score++
		}*/
	}

	if len(role.ParamRegular) > 0 && len(req.Form) > 0 {
		// 参数匹配

		switch role.ParamMode {
		case 0:
			total += len(role.ParamRegular)
			//and匹配
			hits := 0
			for k, v := range role.ParamRegular {
				if hitJudge(v.Value, req.Form.Get(k), v.Operation, v.Attach, v.S) {
					hits++
				}
			}
			if hits == len(role.ParamRegular) {
				score++
			}
			break
		case 1:
			total++
			//or匹配
			for k, v := range role.ParamRegular {
				if hitJudge(v.Value, req.Form.Get(k), v.Operation, v.Attach, v.S) {
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
	if role.f == nil {
		fmt.Println("role:", role.Id, " f() is null")
		return "", false
	}

	return "http://" + role.f() + req.URL.String(), true
}

func hitJudge(target, original, operation string, attach []string, s map[string]interface{}) bool {
	var tmp = original
	for _, a := range attach {
		switch a {
		case MOD:
			if v, ok := s[a]; ok {
				n, err := strconv.Atoi(tmp)
				if err != nil {
					panic("ERROR: Mold operation of attach, original param is not number")
				}
				safeInt, err := util.SafeInt(v)
				if err != nil {
					//
					panic("ERROR: Mold operation of attach, mod param is not number")
				}
				n = n % safeInt
				tmp = strconv.Itoa(n)
				break
			}
			panic("Modulo operation of attach is not divisible")
			break
		case HASH:
			hashcode := util.HashCode(tmp)
			if hashcode == 0 {
				panic("hashcode is zero, string of" + tmp)
			}
			tmp = strconv.Itoa(hashcode)
			break
		default:
			panic("no attach model.")
		}
	}

	switch operation {
	case LT:
		if tmp < target {
			return true
		}
		break
	case GT:
		if tmp > target {
			return true
		}
		break
	case EQUAL:
		if strings.Compare(tmp, target) == 0 {
			return true
		}
		break
	default:
		panic("no operation model.")
	}
	return false

}
