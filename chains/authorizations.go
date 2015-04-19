// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//

package chains

import (
	"bytes"
	"encoding/json"
	mrand "math/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/letsencrypt/boulder/core"	
	"github.com/letsencrypt/boulder/jose"	
)

var pollThrottle time.Duration = time.Miliseccond * 100

// random sprinkling of TLDs from boulder/policy/psl.go
var TLDs []string = []string{
	".com",
	".co.uk",
	".net",
	".org",
	".edu",
	"ac.ae",
	"ac.at",
	"ac.be",
	"ac.ci",
	"ac.cn",
	"ac.cr",
	"ac.gn",
	"ac.id",
	"ac.im",
	"ac.in",
	"ac.ir",
	"ac.jp",
	"ac.kr",
	"ac.ma",
	"ac.me",
	"ac.mu",
	"ac.mw",
	"ac.nz",
	"ac.pa",
	"ac.pr",
	"ac.rs",
	"ac.ru",
	"ac.rw",
	"ac.se",
	"ac.sz",
	"ac.th",
	"ac.tj",
	"ac.tz",
	"ac.ug",
	"ac.uk",
	"ac.vn",
	"panama.museum",
	"panerai",
	"parachuting.aero",
	"paragliding.aero",
	"paris",
	"paris.museum",
	"parliament.nz",
	"parma.it",
	"paroch.k12.ma.us",
	"pars",
	"parti.se",
	"partners",
	"parts",
	"party",
	"pasadena.museum",
	"pavia.it",
	"pb.ao",
	"pc.it",
	"pc.pl",
	"varoy.no",
	"vb.it",
	"vc",
	"vc.it",
	"vda.it",
	"vdonsk.ru",
	"ve",
	"ve.it",
	"vefsn.no",
	"vega.no",
	"vegarshei.no",
	"vegas",
	"ven.it",
	"veneto.it",
	"venezia.it",
	"venice.it",
	"vennesla.no",
	"ventures",
	"verbania.it",
	"vercelli.it",
	"verdal.no",
	"verona.it",
	"verran.no",
	"versailles.museum",
	"versicherung",
	"org.ng",
	"org.nr",
	"org.nz",
	"org.om",
	"org.pa",
	"org.pe",
	"org.pf",
	"org.ph",
	"org.pk",
	"org.pl",
	"org.pn",
	"org.ps",
	"org.pt",
	"org.py",
	"org.qa",
	"org.ro",
	"org.rs",
	"org.ru",
	"org.sa",
	"org.sb",
	"org.sc",
	"org.sd",
	"org.se",
	"org.sg",
	"org.sh",
	"org.sl",
	"org.sn",
	"org.so",
	"org.st",
	"org.sv",
	"org.sy",
	"org.sz",
	"org.tj",
	"org.tm",
	"org.tn",
	"org.to",
	"org.tr",
	"org.tt",
	"org.tw",
	"org.ua",
	"org.ug",
	"org.uk",
	"org.uy",
	"org.uz",
	"org.vc",          
}

var dnsLetters string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

func NewAuthorizationTestChain() (ChainResult) {
	// check that there is at least one registration or return empty CR so attacker can continue
	var cR ChainResult

	if !chains.existingRegistrations() {
		return cR
	}

	cR.Name = "new authorization"

	// generate a random domain name (should come up with some fun names... THE NEXT GOOGLE PERHAPS?)
	var buff bytes.Buffer
	mrand.Seed(time.Now().UnixNano())
	randSuffix := TLDs[mrand.Intn(len(TLDs))]
	randLen := mrand.Intn(61-len(randSuffix))+1
	for i := 0; i < randLen; i++ {
		buff.WriteString(dnsLetters[mrand.Intn(len(dnsLetters))])
	}
	randomDomain := fmt.Sprintf("%s.%s", buff.String(), randSuffix)

	// create the registration object
	initAuth := fmt.Sprintf("{\"identifier\":{\"type\":\"dns\",\"value\":\"%s\"}}", randomDomain)

	// build the JWS object
	alg := jose.RSAPKCS1WithSHA256
	payload := []byte(initAuth)
	jws, err := jose.Sign(alg, TheKey, payload)

	// into JSON
	requestPayload, _ := json.Marshal(jws)
	// send a timed POST request
	client := &http.Client{}
	body, status, _, timing, err := timedPOST(client, "http://localhost:4000/acme/new-authorization", requestPayload)
	var postResult requestResult
	postResult.Uri = "/acme/new-authorization"
	postResult.Took = timing
	if status != 201 {
		// baddy
		postResult.Successful = false
		postResult.Error = "Incorrect status code"
		fmt.Println(status, string(body))

		cR.IndividualResults = append(cR.IndividualResults, postResult)
		cR.Successful = false
		cR.Took = time.Since(chainStart)
		return cR
	}

	var respAuth core.Authorization
	err = json.Unmarshal(body, &respAuth)
	if err != nil {
		// uh...
		postResult.Successful = false
		postResult.Error = err.Error()

		cR.IndividualResults = append(cR.IndividualResults, postResult)
		cR.Successful = false
		cR.Took = time.Since(chainStart)
		return cR
	}

	postResult.Successful = true
	cR.IndividualResults = append(cR.IndividualResults, postResult)

	// pick which challenges we want to do, prefer simpleHttps and Dvsni...

	// setup challenge stuffffff :>
	for _, chall := range respAuth.Challenges {
		select chall.Type {
		case "simpleHttps":
			chains.SimpleHTTPSChalls[randomDomain] = chall.Token

			chall.Path = ""
		case "dvsni":
			S := sha256.Sum256(randomDomain)
			RS := append(R, S...)
			Z := fmt.Sprintf("%x", sha256.Sum256(RS))

			chains.DvsniChalls[chall.Nonce] = Z

			chall.S = S
		}

		// send updated chall object!
		challJson, _ := json.Marshal(chall)
		payload = []byte(challJson)
		jws, _ = jose.Sign(alg, TheKey, payload)

		// send a timed POST request
		body, status, _, timing, err := timedPOST(client, chall.Uri, requestPayload)
		var updateResult requestResult
		updateResult.Uri = "/acme/authz/update-challenge"
		updateResult.Took = timing
		if status != 201 {
			// baddy
			updateResult.Successful = false
			updateResult.Error = "Incorrect status code"
			fmt.Println(status, string(body))

			cR.IndividualResults = append(cR.IndividualResults, updateResult)
			cR.Successful = false
			cR.Took = time.Since(chainStart)
			return cR
		}

		// ignore updated auth json obj...
		updateResult.Successful = true
		cR.IndividualResults = append(cR.IndividualResults, updateResult)
	}

	// check chains.DvsniChalls and chains.SimpleHttpsChalls until key has been deleted
	// or just say fuck it and start polling immediately ^_^
	var totalTiming time.Duration
	var pollResult requestResult
	pollResult.Uri = "/acme/authz/poll-authz"
	for {
		// i guess that answers that question...
		body, status, _, timing, err := timedGet(client, "http://localhost:4000/acme/new-reg", requestPayload)
		totalTiming += timing
		if status != 200 {
			// baddy
			pollResult.Successful = false
			pollResult.Error = "Incorrect status code"
			pollResult.Timing = totalTiming
			fmt.Println(status, string(body))

			cR.IndividualResults = append(cR.IndividualResults, pollResult)
			cR.Successful = false
			cR.Took = time.Since(chainStart)
			return cR
		}

		var pollAuth core.Authorization
		err = json.Unmarshal(body, &pollAuth)
		if err != nil {
			// uh...
			pollResult.Successful = false
			pollResult.Error = err.Error()
			pollResult.Timing = totalTiming

			cR.IndividualResults = append(cR.IndividualResults, pollResult)
			cR.Successful = false
			cR.Took = time.Since(chainStart)
			return cR
		}

		if pollAuth.Status == "valid" {
			// we are are a freeeeeee loop
			break
		}

		time.Sleep(pollThrottle)
	}
	pollResult.Took = totalTiming
	pollResult.Successful = true
	cR.IndividualResults = append(cR.IndividualResults, pollResult)

	// not yet!
	// cR.Successful = true
	// cR.Took = time.Since(chainStart)

	return cR
}