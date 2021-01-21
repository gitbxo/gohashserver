package main

import (
    "crypto/sha512"
    "encoding/base64"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
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
var hashMap = make(map[string]string)
var serverPort = "8080"
var sleepSeconds = time.Duration(5)


func hashGET(w http.ResponseWriter, req *http.Request) {
    key := strings.TrimPrefix(req.URL.Path, "/hash/")
    val, ok := hashMap[key]
    if ok {
        fmt.Fprintf(w, "%s\n", val)
    } else {
        fmt.Fprintf(w, "Missing index %s\n", key)
    }
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
    time.Sleep(sleepSeconds * time.Second)
    sha_512 := sha512.New()
    sha_512.Write([]byte(value))
    hashVal := base64.StdEncoding.EncodeToString(sha_512.Sum(nil))
    hashMap[strconv.FormatInt(key, 10)] = hashVal
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


// This handles POST requests for /hash
//
// When the body includes the password parameter,
//      it calls hashPOST to time the call and save the hash value
// If the password form value is blank or missing, it does nothing
//
// If the method is not POST, it returns 404 (Http Not Found)
//
func hashHandler(w http.ResponseWriter, req *http.Request) {
    start := time.Now()

    if req.Method == "POST" {
        password := req.PostFormValue("password")
        if len(password) > 0 {
            hashPOST(w, start, password)
        }
    } else {
        http.NotFound(w, req)
    }
}


func processArgs() {
    argName := ""
    for i := 1; i < len(os.Args); i++ {
        if argName == "-port" {
            portVal, portErr := strconv.ParseInt(os.Args[i], 10, 64)
            if portErr == nil {
                if portVal > 1 {
                    serverPort = os.Args[i]
                }
            }
        }
        if argName == "-sleep" {
            sleepVal, sleepErr := strconv.ParseInt(os.Args[i], 10, 64)
            if sleepErr == nil {
                if sleepVal > 0 {
                    sleepSeconds = time.Duration(sleepVal)
                }
            }
        }
        argName = os.Args[i]
    }
}


func main() {
    processArgs()

    http.HandleFunc("/hash", hashHandler)
    http.HandleFunc("/hash/", hashGET)
    http.HandleFunc("/stats", statsHandler)
    http.ListenAndServe(":" + serverPort, nil)
}
