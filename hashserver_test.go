package main

import (
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)


func TestHashHandler_GET_404(t *testing.T) {

    r, err := http.NewRequest("GET", "/hash", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()
    hashHandler(w, r)

    res := w.Result()

    if res.StatusCode != http.StatusNotFound {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
}


// When no password is provided, counter is not incremented
func TestHashHandler_POST_nil_ignored(t *testing.T) {

    r, err := http.NewRequest("POST", "/hash", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()

    priorVal := hashCounter.c
    hashHandler(w, r)
    postVal := hashCounter.c

    res := w.Result()

    if postVal != priorVal {
        t.Errorf("Counter found %d expected %d", postVal, priorVal)
    }
    if res.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
}


func TestHashGET_999_404(t *testing.T) {

    r, err := http.NewRequest("GET", "/hash/999", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()
    hashGET(w, r)

    res := w.Result()

    if res.StatusCode != http.StatusNotFound {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
}


func TestHashGET_angryMonkey(t *testing.T) {

    r, err := http.NewRequest("GET", "/hash/1", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()
    saveHashValue(1, "angryMonkey")
    hashGET(w, r)

    res := w.Result()

    if res.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
    if w.Body.String() != "ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==\n" {
        t.Errorf("Unexpected result body %s", w.Body.String())
    }
}


// When password is provided, counter is incremented
func TestHashPOST_increment(t *testing.T) {

    start := time.Now()
    w := httptest.NewRecorder()

    priorVal := hashCounter.c
    hashPOST(w, start, "abc")
    postVal := hashCounter.c

    res := w.Result()

    if postVal != (priorVal + 1) {
        t.Errorf("Counter found %d expected %d", postVal, (priorVal + 1))
    }
    if res.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
}


func TestStatsHandler_nonzero(t *testing.T) {

    r, err := http.NewRequest("GET", "/stats", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()

    requests := hashCounter.c
    totalTime := hashTime.c
    statsHandler(w, r)

    res := w.Result()

    if res.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
    if requests == 0 {
        t.Errorf("Unexpected request count %d", requests)
    } else {
        expected := fmt.Sprintf("{\"total\": %d, \"average\": %d}\n", requests, (totalTime / requests))
        if (w.Body.String() != expected) {
            t.Errorf("Unexpected result body %s", w.Body.String())
        }
    }
}


func TestStatsHandler_zero(t *testing.T) {

    r, err := http.NewRequest("GET", "/stats", nil)
    if err != nil {
        t.Fatal(err)
    }
    w := httptest.NewRecorder()

    requests := hashCounter.c
    hashCounter.Add(-requests)
    statsHandler(w, r)

    res := w.Result()

    if res.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code %d", res.StatusCode)
    }
    if w.Body.String() != "{\"total\": 0, \"average\": 0}\n" {
        t.Errorf("Unexpected result body %s", w.Body.String())
    }
}

