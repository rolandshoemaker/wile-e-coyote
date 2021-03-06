// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//
// This is a load tester for the Boulder CA server!

package main

import  (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/cactus/go-statsd-client/statsd"
	_ "github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/go-sql-driver/mysql"

	"github.com/rolandshoemaker/wile-e-coyote/chains"
)

var version string = "0.0.1"

var statsdServer string = "localhost:8125"

var sqlDriver string = "mysql"
var sqlServer string = "root:password@tcp(192.168.125.2:3306)/boulder"

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

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &chains.TheKey.PublicKey, &chains.TheKey)
	if err != nil {
		log.Println("Couldn't create cert:", err)
		return &tls.Certificate{}
	}

	cert = &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey: &chains.TheKey,
	}
	return
}

var acmeSuffix string = ".acme.invalid"

// creates the right certificate with relevant DNS names and public key based on provided ServerName
// (from SNI).
func getCert(clientHello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	var dnsNames []string
	if strings.HasSuffix(clientHello.ServerName, acmeSuffix) {
		// DVSNI challenge so lets compute Z... FUN
		nonce := clientHello.ServerName[0:len(clientHello.ServerName)-len(acmeSuffix)]

		// assume NewAuthorizationTestChain() has set this already and clean it up
		Dvsni := chains.DvsniChalls[nonce]
		delete(chains.DvsniChalls, nonce)
		
		dnsNames = append(dnsNames, Dvsni.Domain, fmt.Sprintf("%s%s", Dvsni.Z, acmeSuffix))
	} else {
		dnsNames = []string{clientHello.ServerName}
	}

	cert = genSelfSigned(dnsNames)
	return
}

func listenAndBeAGenius(srv *http.Server) error {
	//
	//         Wile E. Coyote
	//             GENIUS
	//
	// HAVE BRAIN          WILL TRAVEL
	//
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
	fmt.Println("HIT BINGBINGBING\n\n")
	if !strings.HasSuffix(req.TLS.ServerName, ".acme.invalid") {
		// assume NewAuthorizationTestChain() has set this already and clean it up
		token := chains.SimpleHTTPSChalls[req.TLS.ServerName]
		delete(chains.SimpleHTTPSChalls, req.TLS.ServerName)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(token))
	}
	return
}

func runChallSrv() {
	http.HandleFunc("/", handler)
	httpsServer := &http.Server{Addr: "192.168.125.2:443"}
	fmt.Println("Running Challenge server... [Remember to redirect all DNS A records to the local ip address!]")
	listenAndBeAGenius(httpsServer)
}

//////////////
// Attacker //
//////////////

var numAttackers int = 0

func attacker(closeChan chan bool) {
	stats, err := statsd.NewClient(statsdServer, "Wile-E-Coyote")
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

func ArithmeticRampUp(workerIncrement int, finalWorkers int, timeInterval time.Duration) {
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
		log.Printf("Work period %d, set numAttackers -> %d...\n", i, numAttackers)
		aliveAttackers = monitorHerd(aliveAttackers)
		time.Sleep(timeInterval)
	}
}

func justHammer(numWorkers int) {
	fmt.Println("\n# Starting hammering\n")
	numAttackers = numWorkers
	fmt.Printf("Number of workers: %d\n", numAttackers)

	go runChallSrv()
	//var aliveAttackers []chan bool
	//aliveAttackers = monitorHerd(aliveAttackers)

	// wait around foreverz
	wait := make(chan bool)
	<-wait
}

func plainSeq(workerSeq []int, timeInterval time.Duration) {
	totalDuration := time.Duration(timeInterval.Nanoseconds() * int64(len(workerSeq)))

	fmt.Printf("Worker sequence: %v\n", workerSeq)
	fmt.Printf("Work period length: %s\n", timeInterval)
	fmt.Printf("Num work periods: %d\n", len(workerSeq))
	fmt.Printf("Total test duration: %s\n", totalDuration)

	fmt.Println("\n# Starting sequence test\n")

	go runChallSrv()
	var aliveAttackers []chan bool
	for i, workers := range workerSeq {
		numAttackers = workers
		log.Printf("Work period %d, set numAttackers -> %d...\n", i + 1, numAttackers)
		aliveAttackers = monitorHerd(aliveAttackers)
		time.Sleep(timeInterval)
	}
}

func profile(stats statsd.Statter) {
	for {
		var memoryStats runtime.MemStats
		runtime.ReadMemStats(&memoryStats)

		stats.Gauge("Gostats.Goroutines", int64(runtime.NumGoroutine()), 1.0)

		stats.Gauge("Gostats.Heap.Objects", int64(memoryStats.HeapObjects), 1.0)
		stats.Gauge("Gostats.Heap.Idle", int64(memoryStats.HeapIdle), 1.0)
		stats.Gauge("Gostats.Heap.InUse", int64(memoryStats.HeapInuse), 1.0)
		stats.Gauge("Gostats.Heap.Released", int64(memoryStats.HeapReleased), 1.0)

		gcPauseAvg := int64(memoryStats.PauseTotalNs) / int64(len(memoryStats.PauseNs))

		stats.Timing("Gostats.Gc.PauseAvg", gcPauseAvg, 1.0)
		stats.Gauge("Gostats.Gc.NextAt", int64(memoryStats.NextGC), 1.0)

		time.Sleep(time.Second)
	}
}

func mysqlStats(SQL *sql.DB) (err error) {
	var regs          int
	var pending_authz int
	var authz         int
	var certs         int

	err = SQL.QueryRow("SELECT count(*) FROM registrations").Scan(&regs)
	if err != nil {
		return
	}
	err = SQL.QueryRow("SELECT count(*) FROM pending_authz").Scan(&pending_authz)
	if err != nil {
		return
	}
	err = SQL.QueryRow("SELECT count(*) FROM authz").Scan(&authz)
	if err != nil {
		return
	}
	err = SQL.QueryRow("SELECT count(*) FROM certificates").Scan(&certs)
	if err != nil {
		return
	}

	fmt.Printf("SQL stats\n#########\n\n")
	fmt.Printf("Registrations: %d\n", regs)
	fmt.Printf("Pending authorizations: %d\n", pending_authz)
	fmt.Printf("Authorizations: %d\n", authz)
	fmt.Printf("Certificates: %d\n", certs)
	return
}

var usage string = `wile-e-coyote [subcommand] --mysql MYSQLURI

Subcommands
    hammer  WORKERNUM
            Just hammer the server with a constant worker number.

    seq     INTERVAL WORKERNUM WORKERNUM...
            Increase the number of workers in a fixed sequence with
            fixed interval (in seconds).

    aseq    WORKERINCREMENT FINALWORKERS INTERVAL
            Increase the number of workers in a arithmetic sequence
            with fixed interval (in seconds).

Global Options
    --mysql MYSQLURI    The MySQL URI for the boulder DB (incl. username/password e.g. 
    	                "username:password@tcp(127.0.0.1:3306)/boulder").
	--statsd STATSDURI  The StatsD URI to send all the stuff we collect to.
`

func wecUsage() {
	fmt.Println(usage)
	return
}

func main() {
	fmt.Printf("# wile-e-coyote - a load tester for Boulder [v%s]\n\n", version)
	stats, err := statsd.NewClient(statsdServer, "wile-e-coyote")
	if err != nil {
		log.Println("oh no statsd b0rkd")
		stats, _ = statsd.NewNoopClient(nil)
	}

	go profile(stats)

	if len(os.Args) < 2 {
		wecUsage()
	} else {
		SQL, err := sql.Open(sqlDriver, sqlServer)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = mysqlStats(SQL)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Start time: %s\n", time.Now())
		switch os.Args[1] {
			case "hammer":
				if len(os.Args[1:]) != 2 {
					fmt.Printf("Argument parsing error: Not enough arguments!")
					wecUsage()
					return
				}
				var workers int
				_, err := fmt.Sscanf(os.Args[2], "%d", &workers)
				if err != nil {
					fmt.Printf("Argument parsing error: %s\n", err)
					wecUsage()
					return
				}
				justHammer(workers)
			case "seq":
				if len(os.Args[1:]) < 4 {
					fmt.Printf("Argument parsing error: Not enough arguments!")
					wecUsage()
					return
				}
				var secInterval int
				_, err := fmt.Sscanf(os.Args[2], "%d", &secInterval)
				if err != nil {
					fmt.Printf("Argument parsing error: %s\n", err)
					wecUsage()
					return
				}
				var workSeq []int
				for _, w := range os.Args[3:] {
					var wInt int
					_, err := fmt.Sscanf(w, "%d", &wInt)
					if err != nil {
						fmt.Printf("Argument parsing error: %s\n", err)
						return
					}
					workSeq = append(workSeq, wInt)
				}
				plainSeq(workSeq, time.Duration(secInterval * 1000000000))
			case "aseq":
				if len(os.Args[1:]) != 4 {
					fmt.Printf("Argument parsing error: Not enough arguments!")
					wecUsage()
					return
				}
				var workerInc int
				var finalWorkers int
				var secInterval int
				_, err := fmt.Sscanf(strings.Join(os.Args[2:], " "), "%d %d %d", &workerInc, &finalWorkers, &secInterval)
				if err != nil {
					fmt.Printf("Argument parsing error: %s\n", err)
					return
				}
				ArithmeticRampUp(workerInc, finalWorkers, time.Duration(secInterval * 1000000000))
		}
		err = mysqlStats(SQL)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("End time: %s\n", time.Now())
	}
}
