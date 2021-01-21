package main

import (
    "fmt"
    "net/http"
    "sync/atomic"
    "time"
)

// atomic wrapper around int64
type Int64 struct {
	c int64
}
// create new Int64
func NewInt64(i int64) *Int64 {
	return &Int64{c: i}
}
// add and return value
func (v *Int64) Add(i int64) int64 {
	return atomic.AddInt64(&v.c, i)
}

var hashCounter Int64 = *NewInt64(0)
var hashTime Int64 = *NewInt64(0)
var hashMap = make(map[int64]string)


func hashGET(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintf(w, "hashGET\n")
}


func hashPOST(w http.ResponseWriter, start time.Time, password string) {
    newId := hashCounter.Add(1)
    // Delay saving the hash value
    defer func() { go saveHashValue(newId, password) }()
    fmt.Fprintf(w, "%d\n", newId)

    elapsed := time.Since(start)
    // Add the elapsed time in microseconds
    hashTime.Add(elapsed.Microseconds())
}


func saveHashValue(key int64, value string) {
    // sleep 5 seconds
    time.Sleep(5 * time.Second)
    hashMap[key] = value
}


// For example: curl http://localhost:8080/stats should return:
// {"total": 1, "average": 123}
func statsHandler(w http.ResponseWriter, req *http.Request) {
    requests := hashCounter.c
    // calculate average time in microseconds
    if requests > 0 {
      avgTime := hashTime.c / requests
      fmt.Fprintf(w, "{\"total\": %d, \"average\": %d}\n", requests, avgTime)
    } else {
      fmt.Fprintf(w, "{\"total\": 0, \"average\": 0}\n")
    }
}



func hashHandler(w http.ResponseWriter, req *http.Request) {
    start := time.Now()

    if req.Method == "GET" {
      hashGET(w, req)
    } else {
      if req.Method == "POST" {
        password := req.PostFormValue("password")
        if len(password) > 0 {
          hashPOST(w, start, password)
        }
      } else {
        http.NotFound(w, req)
      }
    }
}


func main() {
    http.HandleFunc("/hash", hashHandler)
    http.HandleFunc("/stats", statsHandler)
    http.ListenAndServe(":8080", nil)
}
