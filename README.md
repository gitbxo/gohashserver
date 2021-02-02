# gohashserver
Go lang server for serving hash values

## Parameters
    port              # http port, defaults to 8080
    hash-delay        # delay for saving hash value, defaults to 5 seconds
    shutdown-timeout  # timeout for shutdown, defaults to 1 minute

## Example command
    go run hashserver.go --port 9000

## API
    POST /hash takes data "password=xyz" and returns hash id
    GET /hash/i returns the hash value for hash id
        Note: POST takes 5 seconds to save value, so will return missing id until value is available
    GET /stats returns number of POST /hash requests and average time in microseconds
    GET /shutdown tells server to shutdown
        Note: Shutdown waits for requests to finish processing
        Note: Interrupt signals also have the same effect as call to /shutdown

## Design
### Graceful Shutdown
    Graceful shutdown is achieved by using sync.WaitGroup objects
    After the shutdown request is received, waits for requests to complete
    All requests increment the wait count and decrement when done

### Delayed Computation
    Asynchronous hash computation is achieved by creating a separate thread
    To start the computation after the request completes, defer is used
    By using defer (to delay starting the computation) and go (separate thread)
    we are able to complete the http request and compute the hash in the background


### Statistics
    The variables for the counter and timer stats are atomic
    This ensures that they are fetched and incremented atomically


## Sample requests
    curl -X POST --data "password=angryMonkey" http://localhost:9000/hash
    ==> 1
    curl -X POST --data "password=secondMonkey" http://localhost:9000/hash
    ==> 2
    curl -X GET http://localhost:9000/stats
    ==> {"total": 2, "average": 32}
    curl -X GET http://localhost:9000/hash/1
    ==> ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==
    curl -X GET http://localhost:9000/hash/2
    ==> fwaxl5tGU2y3NgCH3fbT0uv/jTre6aZh4KQBGoA8cnJe89jGZQO0kiSNqJl6aho+ozFzkPmgz4gM2zx/Iy9MGg==
    curl -X GET http://localhost:9000/shutdown
    ==> Shutting down HTTP server


## Testing
    go test hashserver.go hashserver_test.go


