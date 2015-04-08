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
                                                                     
         ┌────▼─────────────┐ charts with avg. resp timing per test chain
         │wec-analytics tool│ (plus other aggregate information like num 
         └──────────────────┘ users, authorizations, certs, errors etc)  
                                                                     
                                                                
```

[Docker compose](http://docs.docker.com/compose/)+Go based context-aware load testing framework for the `boulder` CA server infrastructure.


### Main components

* `wec-requester` - executes *test chains* as goroutines, each of which makes a series of requests to the `boulder` server, they may or may not retrieve/store information from/in an SQL db, or a `redis` keystore to prevent overlap or facilitate challenge completion where SQL has too much overhead.
* `wec-challenge-server` - listens for TLS connections and automatically generates self-signed certificates and responses to satisfy ACME challenges using information from SQL.
* `wec-analytics tool` - accepts `chainResult`s (and probably other stuff.. `debugResult`?) *probably* via RPC a call, logs them to a file (either JSON or binary for lots of stuff), and also serves a basic webpage displaying metric charts (preeeetty).

### `wec-requester`

Most(/all) of the request `struct`s and JWS/JWK signing methods are already done in `boulder` as well as a whole bunch of super useful utility functions so no need to reinvent the wheel on those fronts.

#### Context-based chain execution

`wec-requester` should randomly pick a test chain to run based on what is available from the SQL db, like so (also there should be settable hard limits for each regs, authz, certs at which the test chains which generate them should no longer be run)

`CHECK: seemingly only certificates can be deleted? (i.e. not authorizations or registrations...)`

* No registrations
  * run `newRegistrationChain`
* Registrations
  * run `newRegistrationChain`
  * run `newAuthorizationChain`
* Registrations + Authorizations
  * run `newRegistrationChain`
  * run `newAuthorizationChain`
  * run `newCertificateChain`
* Registrations + Authorizations + Certificates
  * run `newRegistrationChain`
  * run `newAuthorizationChain`
  * run `newCertificateChain`
  * run `refreshCertificateChain`
  * run `revokeCertificateChain`

#### Possible http libs

For talking to `boulder-wfe`...

* native [net/http](https://godoc.org/net/http) (bit too low level to write quickly, or am I just being lazy...?)
* [goreq](https://github.com/franela/goreq)
* [gorilla/http](http://www.gorillatoolkit.org/pkg/http)

#### *Test chains*

a *test chain* is a complete set of http requests/responses that constitute a single action (i.e. new registration, new authorization etc). each chain should be a method returning a `chainResult` struct containing either the measured metrics or information about an error that occurred that can then be logged in whatever way... (need to come up with this...)

* newRegistrationChain
```
POST        -> /acme/new-reg
SQL [regs]  <- add registration information
```
* newAuthorizationChain
```
POST         -> /acme/new-authorization
SQL [challs] <- add simpleHTTPs path/token (/other challenges...)
POST         -> /acme/authz/asdf/0 (path+token)
GET [poll]   -> /acme/authz/asdf
SQL [auths]  <- add authorization information (priv key and such)
```

etc etc etc...

### Logging

RPC calls from `wec-requester` to `logger`(?) which provides a new `requestResult` struct?

```
type requestResult struct {
    Uri        string        `json:"uri,omitempty"`        // endpoint being hit e.g. "/acme/new-reg"
    Took       time.Duration `json:"took,omitempty"`       // time request actually took
    Successful bool          `json:"successful,omitempty"` // if the *right* thing happened (subjective...)
    Error      string        `json:"error,omitempty"`      // if not right thing, what happd
}

type chainResult struct {
    ChainName         string          `json:"chainname,omitempty"`         // which chain is this result from
    ChainSuccessful   bool            `json:"chainsuccessful,omitempty"`   // was the entire chain successful?
    ChainTook         time.Duration   `json:"chaintook,omitempty"`         // how long did the entire chain take
    IndividualResults []requestResult `json:"individualresults,omitempty"` // individual requestResults
}
```

## Useful references

* [live stats collection](http://nf.id.au/posts/2011/03/collecting-and-plotting-live-data-with-golang.html) - idea for logger...?

