package sniffing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	badServer   = make(map[string]time.Time) //hostname:time
	allServer   = make(map[string]string)    //hostname:url
	noticeGroup = make([]chan map[string]bool, 0, 1)
)

func HealthCheck() {

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("recover：%v", r)
			}
		}()
		context, _ := context.WithTimeout(context.Background(), 2*time.Second)

		for {
			var noticeMap = make(map[string]bool)
			for k, v := range allServer {
				url := "http://" + k + v
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					log.Fatal(err)
				}

				reqTimeout := req.WithContext(context)
				resp, err := http.DefaultClient.Do(reqTimeout)
				if err != nil {
					/*go func() {
						time.AfterFunc(2*time.Second, func() {
							resp, err := http.DefaultClient.Do(reqTimeout)
							if err != nil {
								badServer[k] = time.Now()
							}
							resp.Body.Close()
						})
					}()*/
					//TODO warning the server seems to be not available
					noticeMap[k] = false
					badServer[k] = time.Now()
					continue
				}
				resp.Body.Close()

				if _, ok := badServer[k]; ok {
					delete(badServer, k)
					noticeMap[k] = true
				}
			}
			if len(noticeMap) > 0 {
				for _, c := range noticeGroup {
					if c == nil {
						//chan 为nil直接退出
						break
					}
					c <- noticeMap
				}
			}
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " running health check...")
			time.Sleep(5 * time.Second)
		}
	}()

}
