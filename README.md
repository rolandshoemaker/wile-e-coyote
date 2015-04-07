```
                                                                
                                                                
                            wile e coyote                       
                            ■───────────■                       
                             load tester                        
                                                                
        monolithic client or                                    
      individual containers +                                   
         rabbitmq container                                     
                                                                
                         ┌───────┐           
                         │boulder◀────┐         
                         └───▲───┘    │        
                             │        │        
                             │        │                         
                             │        │     ┌─────────┐         
         JSON   ◀──┐         │        └─────▶ssl proxy│         
       log file    │         │              └────▲────┘         
                   │         │                   │              
           │       │         │                   │              
           │       │   ┌─────▼───────┐    ┌──────▼─────────────┐
           │       └───┤wec-requester│    │wec-challenge-server│
           │           └─▲─────▲─────┘    └───────▲────────────┘
           │                   │                  │             
           │             │     │   ┌─────────┐    │             
           │                   └───▶sql/redis◀────┘             
           │   ─ ─ ─ ─ ─ ┘         └─────────┘                  
           │  │     web ui for                                  
           │    control/live stats?                             
           │  │                                                 
           │                                                    
         ┌─▼──▼─────────┐                                       
         │analytics tool│                                       
         └──────────────┘                                       
       charts with avg. resp timing per test chain              
       (plus other aggregate information like num               
       users, authorizations, certs, errors etc)                
                                                                
```

![wile e coyote](https://33.media.tumblr.com/3df5a7e4f14b272d7a408c18e778a0a8/tumblr_nicwdjwCpO1s2wio8o1_500.gif)
![pokey](http://media.giphy.com/media/52kICijFBOkOQ/giphy.gif)

[Docker compose](http://docs.docker.com/compose/)+Go based context-aware testing framework. Implements multiple *test chains* which can be run concurrently to gather performance metrics about the `boulder` ACME server.

Consisting of two main parts and a whole bunch of aux services.

### main components
* `wec-requester` - executes *test chains* as goroutines, each of which makes a series of requests to the `boulder` server, they may or may not store information in an `sql` db, or a `redis` keystore to prevent overlap or facilitate challenge completion
* `wec-challenge-server` - a basic http server sitting behind a ssl proxy which will automatically generate self-signed certificates required for `simpleHTTPS` challenges and serve challenge tokens from the db based on the domain in the request

### aux components

* *ssl proxy* - sits between `boulder` and `wec-challenge-server` and automatically generates self-signed SSL certs for EVERYTHING
* *sql* / *redis* - `sql` for the registration, authorization, and certificate information, and `redis` for things we are currently working on so other goroutines know to leave us alone. (i.e. don't try to revoke a cert while we are renewing it elsewhere or try to create an authorization for a domain that already exists) `<-` although it should also throw weird stuff at the server as well (prob specific *bad test chains*) for proper load testing...

### test framework

a lot of the annoying stuff we need (JWK, JWS, various marshalable request structs, etcetc) already exist in `boulder` so we can just import a lot of that stuff directly.

which tests are executed should be based on what is in the `sql` db, e.g. if we have no regs run new-registration, if we have registrations run new-registration and new-authorization, if we have registrations and authorizations run new-registration, new-authorization, and new-certificate etc etc etc...

#### possible http libs

* native (bit too low level to write quickly)
* [goreq](https://github.com/franela/goreq)
* [gorilla/http](http://www.gorillatoolkit.org/pkg/http)

#### logging

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
### *test chains*

a *test chain* is a complete set of http requests/responses that constitute a single action (i.e. new registration, new authorization etc). each chain should be a method returning a `chainResult` struct containing either the measured metrics or information about an error that occurred that can then be logged in whatever way... (need to come up with this...)

* new registration
```
POST        -> /acme/new-reg
SQL [regs]  <- add registration information
```
* new-authorization
```
POST         -> /acme/new-authorization
SQL [challs] <- add simpleHTTPs path/token (/other challenges...)
POST         -> /acme/authz/asdf/0 (path+token)
GET [poll]   -> /acme/authz/asdf
SQL [auths]  <- add authorization information
```

### transparent ssl proxy (`squid`)

    openssl genrsa -out squid.key 2048
    openssl req -new -key squid.key -out squid.csr
    openssl x509 -req -days 3650 -in squid.csr -signkey squid.key -out squid.crt
    cat squid.key squid.crt > squid.pem

## useful links

* [live stats collection](http://nf.id.au/posts/2011/03/collecting-and-plotting-live-data-with-golang.html)

