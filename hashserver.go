package main

import (
    "fmt"
    "net/http"
    "sync/atomic"
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


func hashGET(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintf(w, "hashGET\n")
}


func hashPOST(w http.ResponseWriter, req *http.Request) {
    var newId = hashCounter.Add(1)
    // defer saveHashValue(hashValue)
    fmt.Fprintf(w, "hashPOST %d\n", newId)
}


func hashHandler(w http.ResponseWriter, req *http.Request) {
    if req.Method == "GET" {
      hashGET(w, req)
    } else {
      if req.Method == "POST" {
        hashPOST(w, req)
      } else {
        http.NotFound(w, req)
      }
    }
}


func main() {
    http.HandleFunc("/hash", hashHandler)
    http.ListenAndServe(":8090", nil)
}
