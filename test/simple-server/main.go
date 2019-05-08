package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type responseTime struct {
	probalility  int // expressed in %
	responseTime time.Duration
}
type responseCode struct {
	probalility  int // expressed in %
	responseCode int
}

func main() {
	// HOSTNAME be be used in the body reply
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	// Response time piloting
	responseTimeFlags := flag.String("responseTime", "", "ordered comma separated (probability in %:response time in ms)")
	// Response code piloting
	responseCodeFlags := flag.String("responseCode", "", "ordered comma separated (probability in %:response code). If not hit, 200 will be returned")

	flag.Parse()

	responseTimes := []responseTime{}
	if *responseTimeFlags != "" {
		if match, _ := regexp.MatchString("^([0-9]+:[0-9]+,)*([0-9]+:[0-9]+)$", *responseTimeFlags); !match {
			//if match, _ := regexp.MatchString("^([0-9]+:[0-9]+$)", *responseTimeFlags); !match {
			log.Fatalf("Bad format for parameter responseTime")
		}
		for _, rstr := range strings.Split(*responseTimeFlags, ",") {
			rt := responseTime{}
			var ms int
			fmt.Sscanf(rstr, "%d:%d", &rt.probalility, &ms)
			if rt.probalility > 100 || rt.probalility <= 0 {
				log.Fatalf("issue with responseTime parameter with element '%s'. Probability must be in range ]0:100]", rstr)
			}
			rt.responseTime = time.Duration(ms) * time.Millisecond
			responseTimes = append(responseTimes, rt)
		}
	}
	responseCodes := []responseCode{}
	if *responseCodeFlags != "" {
		for _, rstr := range strings.Split(*responseCodeFlags, ",") {
			rt := responseCode{}
			fmt.Sscanf(rstr, "%d:%d", &rt.probalility, &rt.responseCode)
			if rt.probalility > 100 || rt.probalility <= 0 {
				log.Fatalf("issue with responseCode parameter with element '%s'. Probability must be in range ]0:100]", rstr)
			}
			responseCodes = append(responseCodes, rt)
		}
	}

	delayFunc := func() responseTime {
		for _, rt := range responseTimes {
			if rand.Intn(100) < rt.probalility {
				time.Sleep(rt.responseTime)
				return rt
			}
		}
		return responseTime{}
	}
	responseCodeFunc := func() int {
		for _, rt := range responseCodes {
			if rand.Intn(100) <= rt.probalility {
				return rt.responseCode
			}
		}
		return http.StatusOK
	}

	hostHandler := func(w http.ResponseWriter, req *http.Request) {
		delayFunc()
		w.WriteHeader(responseCodeFunc())
		if _, err := io.WriteString(w, hostname); err != nil {
			log.Printf("error writting output: %#v", err)
		}
	}

	delayHandler := func(w http.ResponseWriter, req *http.Request) {
		rt := delayFunc()
		w.WriteHeader(responseCodeFunc())
		fmt.Fprintf(w, "%d ms", int(rt.responseTime.Seconds()*1000))
	}
	http.HandleFunc("/host", hostHandler)
	http.HandleFunc("/delay", delayHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
