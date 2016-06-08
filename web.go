package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type WebObserver struct {
	URL     string
	Timeout time.Duration
	Default int
}

func timeoutDialler(ns time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		c, err := net.Dial(netw, addr)
		if err != nil {
			return nil, err
		}
		if ns.Seconds() > 0.0 {
			c.SetDeadline(time.Now().Add(ns))
		}
		return c, nil
	}
}

func (self *WebObserver) post(data interface{}) int {
	if len(self.URL) == 0 || self.URL == "none" {
		return self.Default
	}
	jdata, err := json.Marshal(data)
	if err != nil {
		return self.Default
	}
	c := http.Client{
		Transport: &http.Transport{
			Dial: timeoutDialler(self.Timeout),
		},
	}
	resp, err := c.Post(self.URL, "application/json", bytes.NewReader(jdata))
	if err != nil {
		return self.Default
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

type electedMessage struct {
	Time int64 `json:"time"`
}

func (self *WebObserver) OnBeingElected() {
	msg := new(electedMessage)
	msg.Time = time.Now().Unix()
	self.post(msg)
}

type WebAPI struct {
	bully    *Bully
	showPort bool
	unixTime bool
}

const (
	newCandidate = "/join"
	getLeader    = "/leader"
)

func NewWebAPI(bully *Bully, showPort, unixTime bool) *WebAPI {
	ret := new(WebAPI)
	ret.bully = bully
	ret.showPort = showPort
	ret.unixTime = unixTime
	return ret
}

func (self *WebAPI) join(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not implemented\r\n")
}

func (self *WebAPI) leader(w http.ResponseWriter, r *http.Request) {
	leader, timestamp, err := self.bully.Leader()
	if err != nil {
		fmt.Fprint(w, "Error: %v\r\n", err)
	}
	var leaderAddr string
	imleader := "remote"
	if self.bully.MyId().Cmp(leader.Id) == 0 {
		imleader = "local"
		if len(leader.Addr) == 0 {
			leaderAddr = self.bully.MyAddr()
		} else {
			leaderAddr = leader.Addr
		}
	} else {
		leaderAddr = leader.Addr
	}

	if !self.showPort {
		ae := strings.Split(leaderAddr, ":")
		if len(ae) > 1 {
			leaderAddr = strings.Join(ae[:len(ae)-1], ":")
		}
	}
	if self.unixTime {
		fmt.Fprintf(w, "%v\t%v\r\n%v\r\n", imleader, leaderAddr, timestamp.Unix())
	} else {
		fmt.Fprintf(w, "%v\t%v\r\n%v\r\n", imleader, leaderAddr, timestamp)
	}
}

func (self *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	switch r.URL.Path {
	case newCandidate:
		self.join(w, r)
	case getLeader:
		self.leader(w, r)
	}
}

func (self *WebAPI) Run(addr string) {
	http.Handle(newCandidate, self)
	http.Handle(getLeader, self)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
