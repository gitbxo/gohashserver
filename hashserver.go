package main

import (
    "context"
    "crypto/sha512"
    "encoding/base64"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
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
    httpShutdown        = &sync.WaitGroup{}
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
        log.Printf(fmt.Sprintf("hashGET: missing index %s", key))
        http.NotFound(w, req)
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

    // sleep for hashDelay
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
    defer httpShutdown.Done()

    timeoutContext, cancelStop := context.WithTimeout(context.Background(), *shutdownTimeout)
    defer cancelStop()

    if err := server.Shutdown(timeoutContext); err != nil {
        log.Fatal(err)
    }
}


func httpInterrupt(server *http.Server) {
    signalChan := make(chan os.Signal, 1)
    signal.Notify(
        signalChan,
        syscall.SIGHUP,  // kill -SIGHUP XXXX
        syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
        syscall.SIGQUIT, // kill -SIGQUIT XXXX
    )

    <-signalChan
    log.Print("os.Interrupt - shutting down...\n")
    go httpStop(server)

    go func() {
        <-signalChan
        log.Fatal("os.Kill - terminating...\n")
    }()
}

func main() {
    flag.Parse()

    log.Printf(fmt.Sprintf("main: starting HTTP server at port %d", *serverPort))
    httpShutdown.Add(1)

    httpMux := http.NewServeMux()
    server := &http.Server{ Addr: fmt.Sprintf(":%d", *serverPort), Handler: httpMux }

    httpMux.HandleFunc("/hash", hashHandler)
    httpMux.HandleFunc("/hash/", hashGET)
    httpMux.HandleFunc("/stats", statsHandler)
    httpMux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Shutting down HTTP server\n")
        go httpStop(server)
    })

    go httpStart(server)

    log.Printf("main: serving http requests")

    // handle interrupt signals
    go httpInterrupt(server)

    // wait for shutdown request
    httpShutdown.Wait()
    log.Printf("main: stopping HTTP server")

    // complete http requests or timeout
    httpTimeout := &sync.WaitGroup{}
    httpTimeout.Add(1)
    go func() {
        // stop if all processing is complete
        defer httpTimeout.Done()

        httpWait.Wait()
        log.Printf("main: completed HTTP requests")
    }()
    go func() {
        // stop after shutdownTimeout
        defer httpTimeout.Done()

        // sleep for shutdownTimeout
        time.Sleep( *shutdownTimeout )
        log.Printf("main: shutdown timeout exceeded")
    }()
    httpTimeout.Wait()
}
