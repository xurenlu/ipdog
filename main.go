package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	device string
	addr   string
	ipHold IPHold
)

type IPHold struct {
	currentMacIp string
	lastReportAt int64
}

func pac(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		//generate proxy pac output
		var output string
		if ipHold.currentMacIp == "" {
			output = `
function FindProxyForURL(url, host)
{
     return "DIRECT;";
}
`
		} else {
			output = `
function FindProxyForURL(url, host)
{
     return "SOCKS ` + ipHold.currentMacIp + `:1080";
}`

		}
		_, err := w.Write([]byte(output))
		if err != nil {
			return
		}
	} else {
		ipHold.currentMacIp = ip
		ipHold.lastReportAt = time.Now().Unix()
		w.Write([]byte("done"))
	}

}
func reportIP(d string) {
	ip, err := GetInterfaceIpv4Addr(d)

	if err != nil {
		fmt.Println(err)
	}

	url := "https://ip4.dev/ip.api?ip=" + ip
	var req *http.Request
	if req, err = http.NewRequest(http.MethodGet,
		url, nil); err != nil {
		fmt.Println("request create failed")
		return
	}
	req.Header.Set("accept", "application/json")

	var httpClient = http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("request uri %s failed\n", url)
		return
	}
	var buf []byte
	res.Body.Read(buf)
	fmt.Printf("request result:%d,body:%s\n", res.StatusCode, string(buf))
}
func main() {
	flag.StringVar(&addr, "addr", ":3355", "address that http server listen at")
	flag.StringVar(&device, "device", "en0", "the wifi-device to retrieve IP")
	go func() {
		for {
			reportIP(device)
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		for {
			if time.Now().Unix() > (ipHold.lastReportAt + 20) {
				ipHold.currentMacIp = ""
			}
			time.Sleep(2 * time.Second)
		}
	}()
	http.HandleFunc("/ip", pac)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		return
	}

}

func GetInterfaceIpv4Addr(interfaceName string) (addr string, err error) {
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IP
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}
	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}
	for _, addr := range addrs { // get ipv4 address
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break
		}
	}
	if ipv4Addr == nil {
		return "", errors.New(fmt.Sprintf("interface %s don't have an ipv4 address\n", interfaceName))
	}
	return ipv4Addr.String(), nil
}
