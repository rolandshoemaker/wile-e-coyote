package main

import  (
	"github.com/rolandshoemaker/wile-e-coyote/requester/chains"
)

func runChain(fn func) {
	// run the chain and get the result
	chainResult := fn()

	// write the result to some kind of log... (RPC calls to log server(?) that also displays the charts n stuff?)
}

func main() {

}
