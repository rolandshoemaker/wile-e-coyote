// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//
// This is a load tester for the Boulder CA server!

package main

import  (
	"fmt"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"log"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/cactus/go-statsd-client/statsd"

	"github.com/rolandshoemaker/wile-e-coyote/chains"
)

var statsdServer string = "localhost:8125"
var mysqlServer  string = ""

//////////////////////
// Challenge server //
//////////////////////

func genSelfSigned(dnsNames []string) (cert *tls.Certificate) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1337),
		Subject: pkix.Name{
			Organization: []string{"wile e coyote llc"},
		},
		NotBefore: time.Now(),
		NotAfter: time.Now().AddDate(0, 0, 1),

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: dnsNames,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, chains.TheKey.PublicKey, chains.TheKey)
	if err != nil {
		log.Println("Couldn't create cert:", err)
		return &tls.Certificate{}
	}

	cert = &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey: chains.TheKey,
	}
	return
}

var acmeSuffix string = ".acme.invalid"

// creates the right certificate with relevant DNS names and public key based on provided ServerName
// (from SNI).
func getCert(clientHello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	var dnsNames []string
	//var publicKey string
	if strings.HasSuffix(clientHello.ServerName, acmeSuffix) {
		// DVSNI challenge so lets compute Z... FUN
		nonce := clientHello.ServerName[0:len(clientHello.ServerName)-len(acmeSuffix)]

		// assume NewAuthorizationTestChain() has set and will delete this already
		Dvsni := chains.DvsniChalls[nonce]
		
		dnsNames = append(dnsNames, Dvsni.Domain, fmt.Sprintf("%s%s", Dvsni.Z, acmeSuffix))
	} else {
		dnsNames = []string{clientHello.ServerName}
	}

	cert = genSelfSigned(dnsNames)
	return
}

func listenAndMeepMeep(srv *http.Server) error {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*genSelfSigned([]string{"roadrunner"})}, // because Listener requires at least one
		ClientAuth: tls.NoClientCert,                                            // and it'll never actually be served
		GetCertificate: getCert,
		NextProtos: []string{"http/1.1"},
	}

	conn, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Println("couldn't listen:", err)
	}

	tlsListener := tls.NewListener(conn, tlsConfig)
	return srv.Serve(tlsListener)
}

func handler(w http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.TLS.ServerName, ".acme.invalid") {
		// assume NewAuthorizationTestChain() has set and will delete this already
		token := chains.SimpleHTTPSChalls[req.TLS.ServerName]

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(token))
	}
}

func runChallSrv() {
	http.HandleFunc("/", handler)
	httpsServer := &http.Server{Addr: ":443"}
	fmt.Println("Running Challenge server... [Remember to redirect all DNS A records to the local ip address!]")
	listenAndMeepMeep(httpsServer)
}

//////////////
// Attacker //
//////////////

var numAttackers int = 0

func attacker(closeChan chan bool) {
	stats, err := statsd.NewClient(statsdServer, "wile-e-coyote")
	if err != nil {
		log.Println("oh no statsd b0rkd")
		stats, _ = statsd.NewNoopClient(nil)
	}

	for {
		select {
		case <- closeChan:
			// goodbye cruel world
			break
		default:
			testChain := chains.GetChain()
			chainResult := testChain()
			//if chainResult != chains.ChainResult{} {
				go chains.SendStats(stats, chainResult)
				fmt.Println(chainResult)
			//}
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
	log.Printf("herding, alive: %d, should be alive: %d", numAlive, numAttackers)
	if numAttackers != numAlive {
		if numAttackers < numAlive {
			// randomly kill some attackers when they finish doing their thing...
			// idk why randomly...
			mrand.Seed(time.Now().UnixNano())
			for i := 0; i < (numAlive-numAttackers); i++ {
				randInt := mrand.Intn(numAlive-1)
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

	return alive
}

func rampUp(workerIncrement int, finalWorkers int, timeInterval time.Duration) {
	workPeriods := finalWorkers / workerIncrement
	totalDuration := time.Duration(timeInterval.Nanoseconds() * int64(workPeriods))

	fmt.Printf("Final workers: %d\n", finalWorkers)
	fmt.Printf("Work period length: %s\n", timeInterval)
	fmt.Printf("Num work periods: %d\n", workPeriods)
	fmt.Printf("Total test duration: %s\n", totalDuration)

	fmt.Println("\n# Starting ramping test\n")

	go runChallSrv()
	var aliveAttackers []chan bool
	for i := 1; i <= workPeriods; i++ {
		numAttackers = i * workerIncrement
		log.Printf("Work period %d, setting numAttackers -> %d...\n", i, numAttackers)
		aliveAttackers = monitorHerd(aliveAttackers)
		time.Sleep(timeInterval)
	}
}

func justHammer(numWorkers int) {
	fmt.Println("\n# Starting hammering\n")
	numAttackers = numWorkers
	fmt.Printf("Number of workers: %d\n", numAttackers)

	go runChallSrv()
	var aliveAttackers []chan bool
	aliveAttackers = monitorHerd(aliveAttackers)

	// wait around foreverz
	wait := make(chan bool)
	<-wait
}

func main() {
	
}
