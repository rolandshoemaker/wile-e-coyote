# `wile-e-coyote`

<p align="center"><img src="http://media.giphy.com/media/52kICijFBOkOQ/giphy.gif" /><br>...but with <b>LOTS OF STICKS</b></p>

```
                                                                     
                                                                     
                             load tester                             
                            ■───────────■                            
                                                                     
                                                                     
                   monolithic client or                              
                 individual containers +                             
                    rabbitmq container                               
                                                                     
                         ┌───────┐         ┌───────┐                 
                         │boulder◀───┬ ─ ─ ▶dnsmasq│                 
                         └───▲───┘   │     └───────┘                 
                             │       │  A IN *. -> challenge             
                             │       │        server                 
                             │       │                               
                             │       │                               
                             │       └────────┐                      
                             │                │                      
                             │                │                      
                       ┌─────▼───────┐    ┌───▼────────────────┐     
                       │wec-requester│    │wec-challenge-server│     
                       └─▲─────▲─────┘    └───────▲────────────┘     
      RPC for passing          │                  │                  
     results to logger?  │     │   ┌─────────┐    │                  
                               └───▶sql/redis◀────┘                  
               ─ ─ ─ ─ ─ ┘         └─────────┘                       
              │     web ui for                                       
                control/live stats?                                  
              │                                                      
                                                                     
         ┌────▼────────┐ charts with avg. resp timing per test chain
         │wec-analytics│ (plus other aggregate information like num 
         └─────────────┘ users, authorizations, certs, errors etc)  
                                                                     
                                                                
```

## `TODO`

* `wec-requester` (in `requester/`) - **WIP** (pretty much all the infrastructure needs to be written)
* `wec-analytics` (in `analytics/`) - **WIP** (pretty much all the infrastructure needs to be written)
* `wec-challenge-server` (in `chall-srv/`) - **Pretty much done**

## Design

[Docker compose](http://docs.docker.com/compose/)+Go based context-aware load testing framework for the `boulder` CA server infrastructure.


### Main components

* `wec-requester` - executes *test chains* as goroutines, each of which makes a series of requests to the `boulder` server, they may or may not retrieve/store information from/in an SQL db, or a `redis` keystore to prevent overlap or facilitate challenge completion where SQL has too much overhead.
* `wec-challenge-server` - listens for TLS connections and automatically generates self-signed certificates and responses to satisfy ACME challenges using information from SQL.
* `wec-analytics` - accepts `chainResult`s (and probably other stuff.. `debugResult`?) *probably* via RPC a call, logs them to a file (either JSON or binary for lots of stuff), and also serves a basic webpage displaying metric charts (preeeetty) for the last X timeframe (versus the ENTIRE logs?).

### `wec-requester`

Most(/all) of the request `struct`s and JWS/JWK signing methods are already done in `boulder` as well as a whole bunch of super useful utility functions so no need to reinvent the wheel on those fronts.

#### Control

What is the best way to say 'how hard a load to test'?

* simply add hard control to number of goroutines running
* similar to above, but slowly gradient from X to Y num goroutunes (i.e. `10 -> 5000` over an hour then quit or something?)
* specify 'simulate ~500rps' and automatically adjust number of goroutines based on avg. response time to get closish to desired rps... (vague idea of how this could work)
* allow live change to num goroutines (via `wec-analytics` web interface?)

#### Storage

`wec-requester` will need to store a whole bunch of information (registrations, authorized keys, information about provisioned certificates etc) that it can re-access, information it'll need to pass between goroutines (I'm using X domain no one else use, I'm revoking X cert no one else use, etc), and information it'll need to pass to `wec-challenge-server` (domains expecting challenges and either their token for simpleHTTPS or their `R` and `S` values for Dvsni).

1. For the first case it seems SQL would be the best bet since a lot of the stuff will most likely be accessed multiple times and sit around between uses... [gorm]() seems like a pretty good choice beyond just doing direct SQL statements...

2. For inter-goroutine communication a channel or something seems like a good choice? (I'd also use redis or something for this but I feel like thats just the Python talking...)

3. For challenge data Redis makes the most sense since we'll most likely only need to access it once (or maybe twice?) before we discard it and it should be relatively fast for this application.

#### Possible http libs

For talking to `boulder-wfe`...

* native [net/http](https://godoc.org/net/http) (bit too low level to write quickly, or am I just being lazy...?)
* [goreq](https://github.com/franela/goreq)
* [gorilla/http](http://www.gorillatoolkit.org/pkg/http)

#### Context-based chain execution

`wec-requester` should randomly pick a test chain to run based on what is available from the SQL db, like so (also there should be settable hard limits for each regs, authz, certs at which the test chains which generate them should no longer be run)

`CHECK: seemingly only certificates can be deleted? (i.e. not authorizations or registrations...)`

* No registrations
  * run `NewRegistrationChain`
* Registrations
  * run `NewRegistrationChain`
  * run `NewAuthorizationChain`
* Registrations + Authorizations
  * run `NewRegistrationChain`
  * run `NewAuthorizationChain`
  * run `NewCertificateChain`
* Registrations + Authorizations + Certificates
  * run `NewRegistrationChain`
  * run `NewAuthorizationChain`
  * run `NewCertificateChain`
  * run `RefreshCertificateChain`
  * run `RevokeCertificateChain`

#### *Test chains*

a *test chain* is a complete set of http requests/responses that constitute a single action (i.e. new registration, new authorization etc). each chain should be a method returning a `chainResult` struct containing either the measured metrics or information about an error that occurred that can then be logged in whatever way... (need to come up with this...) Each test chain should be a single file in `requester/chains/` and expose a single public method `...Chain` with `package chains`. (various utility methods are provided in `chains/common.go`)

* NewRegistrationChain
```
POST        -> /acme/new-reg
SQL [regs]  <- add registration information
```
* NewAuthorizationChain
```
POST         -> /acme/new-authorization
SQL [challs] <- add simpleHTTPs path/token (/other challenges...)
POST         -> /acme/authz/asdf/0 (path+token)
GET [poll]   -> /acme/authz/asdf
SQL [auths]  <- add authorization information (priv key and such)
```

etc etc etc...

### `wec-challenge-server`

Simple TLS server based on `net.ListenAndServeTLS` that registers a `GetCert` method for `tls.Config` to automatically generate relevant challenge based certificates and a response handler to satisfy `simpleHTTPS` and `Dvsni` challenge requests made by `boulder`.

`wec-challenge-server` uses Redis to retrieve information about challenges for the requested domain and the public key it should use in the certificate that are provided by `wec-requester`. `wec-challenge-server` should only retrieve values from Redis, `wec-requester` should handle the job of cleaning up things it has stored in Redis at the end of a chain.

Pretty much done at this point ([chall-srv/main.go]()), although it needs to be tested with stuff from redis since I've only tested it manually so far...

### `wec-analytics`

RPC calls from `wec-requester` which provide a new `requestResult` struct?

#### Interface

Should have a basic JS frontend to display charts (i.e. RPS / Avg. resp time vs. Time), table with breakdown of chain result times + request result times (avg. resp times, std error, etc...), number of requester goroutines, current number of regs, auths, certs etc... (highcharts.js or something is prob the best option)

Something like this would be super pretty/awesome...

![](http://urbanairship.com/images/uploads/blog/Screen_Shot_2014-07-11_at_11.57.31_AM.png)

##### Metrics

* RPS
* (total) Avg. resp time
* Avg. resp time by test chain

#### Results

Results should be in the form of a `chainResult` struct containing coarse grained information about the chain execution and a list of `requestResult`s providing fine grained information about individual requests made by the chain.

```
type requestResult struct {
    Uri        string        `json:"uri,omitempty"`        // endpoint being hit e.g. "/acme/new-reg"
    Took       time.Duration `json:"took,omitempty"`       // time request actually took
    Successful bool          `json:"successful,omitempty"` // if the *right* thing happened (subjective...)
    Error      string        `json:"error,omitempty"`      // if not right thing, what went wrong
}

type chainResult struct {
    ChainName         string          `json:"chainname,omitempty"`         // which chain is this result from
    ChainSuccessful   bool            `json:"chainsuccessful,omitempty"`   // was the entire chain successful?
    ChainTook         time.Duration   `json:"chaintook,omitempty"`         // how long did the entire chain take
    IndividualResults []requestResult `json:"individualresults,omitempty"` // individual requestResults
}
```

#### Logging format

`wec-analytics` should generate some log file that will be passed to the Docker host when everything shutdown (or something), containing all of the `chainResults` that were passed to it. This file should probably be in JSON, although if it's going to get REALLY big (like millions of requests) we may want to use a binary logging format...? (in this  case we would then need another tool to extract/convert/whatever the log file afterwards.

## Docker

Using Docker compose simplifies setting up the entire framework both on a single machine and on an actually network for testing (using the invidual Dockerfiles and manual linking etc)

## Useful references

* [live stats collection](http://nf.id.au/posts/2011/03/collecting-and-plotting-live-data-with-golang.html) - idea for logger...?

