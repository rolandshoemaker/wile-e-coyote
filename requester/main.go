package main

import  (
	"fmt"
	"math/rand"
	"time"

	"github.com/rolandshoemaker/wile-e-coyote/requester/chains"
)

var numAttackers int = 25
var results []chains.ChainResult

func attacker(closeChan chan bool) {
	fmt.Println("starting attacker")
	for {
		select {
		case <- closeChan:
			// goodbye cruel world
			break
		default:
			fmt.Println("attack")
			testChain, cC := chains.GetChain()
			chainResult, newContext := testChain(cC)
			chains.UpdateContext(cC, newContext)
			go chains.SendStats(chainResult)
			fmt.Println(chainResult)
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
			rand.Seed(time.Now().UnixNano())
			for i := 0; i < (numAlive-numAttackers); i++ {
				randInt := rand.Intn(numAlive-1)
				randCloseChan := alive[randInt]
				alive = append(alive[:randInt], alive[randInt +1:]...)
				randCloseChan <- true
			}
		} else {
			// start some new attackers 
			for i := 0; i < (numAttackers-numAlive); i++ {
				alive = append(alive, runAttacker())
			}
		}
	}
	fmt.Println("herded", len(alive), numAttackers)

	return alive
}

func main() {
	var aliveAttackers []chan bool

	fmt.Println("um hi")

	//go func() {
	//	fmt.Println("um goroutine")
		//for {
			aliveAttackers = monitorHerd(aliveAttackers)
			wait := make(chan bool)
			<-wait
		//	time.Sleep(time.Second * 5)
		//}
	//}()

	// livin on the edge!
	
}
