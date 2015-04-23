// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//

package chains

import (
	"bytes"
	"encoding/json"
	"crypto/sha256"
	mrand "math/rand"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/letsencrypt/boulder/core"	
	"github.com/letsencrypt/boulder/jose"
)

var pollThrottle time.Duration = time.Millisecond * 100

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

var dnsLetters string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func NewAuthorizationTestChain() (ChainResult) {
	// check that there is at least one registration or return empty CR so attacker can continue
	var cR ChainResult

	if !existingRegistrations() {
		return cR
	}

	cR.Name = "new authorization"
	chainStart := time.Now()

	// generate a random domain name (should come up with some fun names... THE NEXT GOOGLE PERHAPS?)
	var buff bytes.Buffer
	mrand.Seed(time.Now().UnixNano())
	randSuffix := TLDs[mrand.Intn(len(TLDs))]
	randLen := mrand.Intn(61-len(randSuffix))+1
	for i := 0; i < randLen; i++ {
		buff.WriteByte(dnsLetters[mrand.Intn(len(dnsLetters))])
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
	body, status, headers, timing, err := timedPOST(client, "http://localhost:4000/acme/new-authz", requestPayload)
	if err != nil {
		// something
	}
	var postResult requestResult
	postResult.Uri = "/acme/new-authz"
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
		var challField string
		var challData  string
		switch chall.Type {
		case "simpleHttps":
			SimpleHTTPSChalls[randomDomain] = chall.Token

			challField = "path"
			challData = "nop"
		case "dvsni":
			S := sha256.Sum256([]byte(randomDomain))
			R, _ := core.B64dec(chall.R)
			RS := append(R, S[:]...)
			Z := fmt.Sprintf("%x", sha256.Sum256(RS))

			DvsniChalls[chall.Nonce] = DvsniChall{Z: Z, Domain: randomDomain}

			challField = "s"
			challData = core.B64enc(S[:])
		default:
			fmt.Printf("unsupported challenge type: %s\n", chall.Type)
			
			cR.Successful = false
			cR.Took = time.Since(chainStart)
			return cR
		}

		// send updated chall object!
		challJson := fmt.Sprintf("{\"type\":\"%s\",\"%s\":\"%s\"}", chall.Type, challField, challData)
		payload = []byte(challJson)
		jws, _ = jose.Sign(alg, TheKey, payload)

		// send a timed POST request
		challUri := url.URL(chall.URI)
		body, status, _, timing, err := timedPOST(client, challUri.String(), requestPayload)
		if err != nil {
			// something
		}
		var updateResult requestResult
		updateResult.Uri = "/acme/authz/update-challenge-"+chall.Type
		updateResult.Took = timing
		if status != 202 {
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

	// check DvsniChalls and SimpleHttpsChalls until key has been deleted
	// or just say fuck it and start polling immediately ^_^
	var totalTiming time.Duration
	var pollResult requestResult
	pollResult.Uri = "/acme/authz/poll-authz"
	// should this loop have some kind of timeout?
	for {
		// i guess that answers that question...
		body, status, _, timing, err := timedGET(client, headers["Location"][0])
		if err != nil {
			// something
		}
		totalTiming += timing
		if status != 200 {
			// baddy
			pollResult.Successful = false
			pollResult.Error = "Incorrect status code"
			pollResult.Took = totalTiming
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
			pollResult.Took = totalTiming

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

	// WE ARE DONE! WOOHOO
	pollResult.Took = totalTiming
	pollResult.Successful = true
	cR.IndividualResults = append(cR.IndividualResults, pollResult)

	cR.Successful = true
	cR.Took = time.Since(chainStart)

	return cR
}