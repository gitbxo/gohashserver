package main

import (
    "context"
    "crypto/sha512"
    "encoding/base64"
    "flag"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
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


var (
    hashCounter Int64   = *NewInt64(0)
    hashTime Int64      = *NewInt64(0)
    hashMap             = make(map[string]string)

    serverPort          = flag.Int("port", 8080, "http port to listen on")
    httpWait            = &sync.WaitGroup{}
    hashDelay           = flag.Duration("hash-delay", 5 * time.Second,
            "Delay for hash computation (default 5s)")
    shutdownTimeout     = flag.Duration("shutdown-timeout", 1 * time.Minute,
            "Timeout for shutdown (default 1m)")
)


func hashGET(w http.ResponseWriter, req *http.Request) {
    httpWait.Add(1)
    defer httpWait.Done()

    key := strings.TrimPrefix(req.URL.Path, "/hash/")
    val, ok := hashMap[key]
    if ok {
        fmt.Fprintf(w, "%s\n", val)
    } else {
        fmt.Fprintf(w, "Missing index %s\n", key)
    }
}


func hashPOST(w http.ResponseWriter, start time.Time, password string) {
    httpWait.Add(1)
    defer httpWait.Done()

    newId := hashCounter.Add(1)
    // Delay saving the hash value until after request returns
    // This creates a background thread to sleep, then save the hash value
    defer func() { go saveHashValue(newId, password) }()
    fmt.Fprintf(w, "%d\n", newId)

    elapsed := time.Since(start)
    // Add the elapsed time in microseconds
    hashTime.Add(elapsed.Microseconds())
    log.Printf(fmt.Sprintf("hashPost: elapsed %d", elapsed.Microseconds()))
}


func saveHashValue(key int64, value string) {
    httpWait.Add(1)
    defer httpWait.Done()

    // sleep 5 seconds
    time.Sleep( *hashDelay )
    sha_512 := sha512.New()
    sha_512.Write([]byte(value))
    hashVal := base64.StdEncoding.EncodeToString(sha_512.Sum(nil))
    hashMap[strconv.FormatInt(key, 10)] = hashVal
    log.Printf(fmt.Sprintf("saveHashValue: saved %d", key))
}


// For example: curl http://localhost:8080/stats should return:
// {"total": 1, "average": 123}
func statsHandler(w http.ResponseWriter, req *http.Request) {
    httpWait.Add(1)
    defer httpWait.Done()

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
    httpWait.Add(1)
    defer httpWait.Done()

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


func httpStart(server *http.Server) {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}


func httpStop(server *http.Server) {
    timeoutContext, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
    defer cancel()

    if err := server.Shutdown(timeoutContext); err != nil {
        log.Fatal(err)
    }
}


func main() {
    flag.Parse()

    log.Printf(fmt.Sprintf("main: starting HTTP server at port %d", *serverPort))

    httpShutdown := &sync.WaitGroup{}
    httpShutdown.Add(1)

    httpMux := http.NewServeMux()
    server := &http.Server{ Addr: fmt.Sprintf(":%d", *serverPort), Handler: httpMux }

    httpMux.HandleFunc("/hash", hashHandler)
    httpMux.HandleFunc("/hash/", hashGET)
    httpMux.HandleFunc("/stats", statsHandler)
    httpMux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Shutting down HTTP server\n")
        go httpStop(server)
        httpShutdown.Done()
    })

    go httpStart(server)

    log.Printf("main: serving http requests")

    // wait for shutdown request
    httpShutdown.Wait()
    log.Printf("main: stopping HTTP server")

    // wait for shutdown request
    httpWait.Wait()

    log.Printf("main: completed HTTP requests")

}
