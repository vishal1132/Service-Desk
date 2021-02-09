package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
)

type company struct {
	NAgents    int    `json:"Agents"`
	Slots      []int  `json:"slots"`
	CompanyID  string `json:"companyID"`
	LogoutTime int    `json:"logoutTime"`
}

type handler struct {
	l  *zerolog.Logger
	rc *redis.Client
}

type ticket struct {
	CompanyID string `json:"companyID"`
	Priority  string `json:"priority"`
}

var mutex = sync.Mutex{}

var companyMap = map[string]*company{}

// Priority is the type for priority of tickets
type Priority string

const (
	// GOLD Priority
	GOLD Priority = "gold"

	//SILVER Priority
	SILVER Priority = "silver"

	//BRONZE Priority
	BRONZE Priority = "bronze"
)

func (h *handler) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func (h *handler) handleRUOK(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "imok")
}

func (h *handler) handleRegisterCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		c := company{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		if err = json.Unmarshal(body, &c); err != nil {
			log.Println(err)
			return
		}
		var response = map[string]string{}
		if err = registercompany(&c); err != nil {
			response["status"] = "fail"
			resJSON(w, 400, response)
			return
		}
		response["status"] = "success"
		resJSON(w, 400, response)
		return
	}

}

func (h *handler) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		t := ticket{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		if err = json.Unmarshal(body, &t); err != nil {
			log.Println(err)
			return
		}
		success := createTicket(&t)
		var response = map[string]string{}
		if success < 0 {
			response["status"] = "failed"
			resJSON(w, 400, response)
			return
		}
		response["status"] = "success"
		resJSON(w, 400, response)
		return
	}
}

func (h *handler) handleRegisterAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		c := company{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		if err = json.Unmarshal(body, &c); err != nil {
			log.Println(err)
			return
		}
		var response = map[string]string{}
		success := registerAgents(c.CompanyID, c.NAgents)
		if success < 0 {
			response["status"] = "failed"
			resJSON(w, 400, response)
			return
		}
		response["status"] = "success"
		resJSON(w, 400, response)
		return
	}
}

func resJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		fmt.Fprintf(w, "%s", err.Error())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func saveRedis(key string, value interface{}) {

}

func registercompany(c *company) error {
	if len(c.Slots) == 0 {
		c.Slots = make([]int, 480)
	}
	companyMap[c.CompanyID] = c
	return nil
}

func createTicket(t *ticket) int {
	company, ok := companyMap[t.CompanyID]
	if !ok {
		return -1
	}
	minsremaining := (company.LogoutTime-time.Now().Hour())*60 + (60 - time.Now().Minute())
	if minsremaining < 0 {
		return -1
	}
	var i int
	switch Priority(strings.ToLower(t.Priority)) {
	case GOLD:
		i = min(120, int(minsremaining))
	case SILVER:
		i = min(240, int(minsremaining))
	case BRONZE:
		i = min(480, int(minsremaining))
	}
	for ; i > 0; i-- {
		mutex.Lock()
		if company.Slots[i] < company.NAgents {
			company.Slots[i]++
			mutex.Unlock()
			return i
		}
	}
	return -1
}

func pollTicket(companyID string) int {
	company, ok := companyMap[companyID]
	if !ok {
		return -1
	}
	for i := 0; i < len(company.Slots); i++ {
		mutex.Lock()
		if company.Slots[i] > 0 {
			mutex.Unlock()
			return i
		}
	}
	return -1
}

func registerAgents(companyID string, n int) int {
	_, ok := companyMap[companyID]
	if !ok {
		return -1
	}
	mutex.Lock()
	companyMap[companyID].NAgents += n
	mutex.Unlock()
	return 0
}

func closeTicket(companyID string, i int) {
	_, ok := companyMap[companyID]
	if !ok {
		return
	}
	time.Sleep(time.Minute)
	mutex.Lock()
	companyMap[companyID].Slots[i]--
	mutex.Unlock()
}

func createworker(companyID string) {
	go func() {
		for {
			i := pollTicket(companyID)
			if i > 0 {
				closeTicket(companyID, i)
			}
		}
	}()
}

func createworkers(c *company) {
	for i := 0; i < c.NAgents; i++ {
		createworker(c.CompanyID)
	}
}

func moveTickets(comp *company) {
	go func() {
		time.Sleep(time.Minute)
		mutex.Lock()
		for i := 1; i < 479; i++ {
			comp.Slots[i] = comp.Slots[i+1]
		}
		mutex.Unlock()
	}()
}

func agents() {
	for key := range companyMap {
		createworkers(companyMap[key])
	}

	for key := range companyMap {
		moveTickets(companyMap[key])
	}
}
