# `wile-e-coyote`

<p align="center"><img src="http://media.giphy.com/media/52kICijFBOkOQ/giphy.gif" /><br>...but with <b>LOTS OF STICKS</b></p>

```
                                                             
                                                             
                             load tester                     
                            ■───────────■                    
                                                             
                                                             
          monolithic client or polylithic                    
          clients in containers + rabbitmq                   
           container, and mysql somewhere                    
                                                             
                         ┌───────┐         ┌───────┐         
                         │boulder◀───┬ ─ ─ ▶dnsmasq│         
                         └───▲───┘   │     └───────┘         
                             │       │   *. -> challenge     
           ┌────────┐        │       │        server         
           │ statsd │        │       │                       
           └───▲────┘        │       │                       
               │             ├─┐     └─────────────┐         
               │             │ ├────┬───┐          │         
               │             │ │    │   │          │         
               │             │ │    │   │          │         
               │           ┌─┘ │    │   │          │         
         ┌─────┼───────────┼───┼────┼───┼──────────┼────────┐
         │wile-e-coyote ┌──┴───▼─┐┌─┴───▼──┐   ┌───▼─────┐  │
         │              │attacker││attacker│   │challenge│  │
         │              └──┬─────┘└─┬──────┘   │ server  │  │
         │              ┌──▼─────┐┌─▼──────┐   └─────────┘  │
         │              │attacker││attacker│                │
         │              └────────┘└────────┘                │
         │                      ...                         │
         │                                                  │
         └──────────────────────────────────────────────────┘
```

## Design

Go based context-aware highly concurrent load testing framework for the [`boulder`](https://github.com/letsencrypt/boulder) CA server software.

The aim of `wile-e-coyote` is to throughly test `boulder` in a way that will mimic real life user interation with the service via the `WFE`. To accomplish this it executes various test chains in individual goroutines that each mimic a specific set of user actions. The number of goroutines executing these test chains can be controlled by the various test modes the `wile-e-coyote` binary provides.

```bash
$ wile-e-coyote
# wile-e-coyote - a load tester for Boulder [v0.0.1]

wile-e-coyote [subcommand] --mysql MYSQLURI

Subcommands
    hammer  WORKERNUM
            Just hammer the server with a constant worker number.

    seq     INTERVAL WORKERNUM WORKERNUM...
            Increase the number of workers in a fixed sequence with
            fixed interval (in seconds).

    aseq    WORKERINCREMENT FINALWORKERS INTERVAL
            Increase the number of workers in a arithmetic sequence
            with fixed interval (sin seconds).

Global Options
    --mysql MYSQLURI    The MySQL URI for the boulder DB (incl. username/password e.g. 
    	                "username:password@tcp(127.0.0.1:3306)/boulder").
```

The only state `wile-e-coyote` stores itself is information about `simpleHttps` and `Dvsni` challenges that needs to be passed between the `attacker`s and the `challenge server`, the rest of the stuff it needs, information about authorizations and certificates and such, is taken directly from the MySQL database that `boulder` uses as its backing store which should be extremely quick (as long as we don't hit the preformance threshold of MySQL which, you know, shouldn't happen...).

### Modes

#### `hammer`



#### `seq`



#### `aseq`



## Metric collection

`wile-e-coyote` sends the metrics that it collects during test chains to StatsD in seperate goroutines, because StatsD it's pretty awesome, and reduces the time that `wile-e-coyote` spends not-actually-load-testing by outsourcing the collection, averaging, etc of the metrics. Something like `Graphite`+`Grafana` can then be used to visualize the collected stuff (just like `boulder` can!)

### StatsD metrics provided

```
counters
--------
    # Based purely on request / response body size + headers
    [bytes] Wile-E-Coyote.TrafficIn
    [bytes] Wile-E-Coyote.TrafficOut

```

```
gauges
------
    [int] Wile-E-Coyote.NumAttackers
   
```

```
timings
-------
    [nanoseconds] Wile-E-Coyote.ChainsTook.{chain-name}.Successful
    [nanoseconds] Wile-E-Coyote.ChainsTook.{chain-name}.Failed

    [nanoseconds] Wile-E-Coyote.RequestsTook.{endpoint}.Successful
    [nanoseconds] Wile-E-Coyote.RequestsTook.{endpoint}.Failed

```

A bunch of Go profiling metrics (memory usage, number of goroutines, etc etc etc) are also collected under `Gostats.Wile-E-Coyote.` for `wile-e-coyote` itself (not `boulder`).