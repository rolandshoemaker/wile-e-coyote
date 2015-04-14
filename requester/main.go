package main

import  (
	"fmt"
	"math/rand"
	"time"

	"github.com/rolandshoemaker/wile-e-coyote/requester/chains"
)

var numAttackers int = 10
var results []chains.ChainResult

func attacker(closeChan chan bool) {
	fmt.Println("starting attacker")
	for {
		select {
		case <- closeChan:
			// goodbye cruel world
			break
		default:
			testChain := chains.GetChain()
			chainResult := testChain()
			// if empty result wasnt passed...
			if chainResult != ChainResult{} {
				results = append(results, chainResult)
			}
		}
	}
}

func runAttacker() chan bool {
	closeChan := make(chan bool, 1)
	go attacker(closeChan)
	return closeChan
}

func monitorHerd(alive []chan bool) []chan bool {
	numAlive := len(alive)
	fmt.Println("herding", numAlive, numAttackers)
	if numAttackers != numAlive {
		if numAttackers < numAlive {
			// randomly kill some attackers when they finish doing their thing...
			// idk why randomly...
			rand.Seed(time.Now().Unix())
			for i := 0; i < (numAlive-numAttackers); i++ {
				randCloseChan := alive[rand.Intn(numAlive-1)]
				randCloseChan <- true
			}
		} else {
			// start some new attackers 
			for i := 0; i < (numAttackers-numAlive); i++ {
				alive = append(alive, runAttacker())
			}
		}
	}

	return alive
}

func main() {
	var aliveAttackers []chan bool

	fmt.Println("um hi")

	//go func() {
	//	fmt.Println("um goroutine")
		for {
			aliveAttackers = monitorHerd(aliveAttackers)
			time.Sleep(time.Second * 5)
		}
	//}()

	// livin on the edge!
	
}
