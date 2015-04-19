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

var TLDs []string = []string{
	".com",
	".co.uk",
	".net",
	".org",
	".edu",

}

func NewAuthorizationTestChain() (ChainResult) {
	// check that there is at least one registration or return empty CR so attacker can continue
	var cR ChainResult

	cR.Name = "new authorization"

	// generate a random domain name
	var buff bytes.Buffer
	mrand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		buff.WriteString(fmt.Sprintf("%d", mrand.Intn(9)))
	}
	randomDomain := buff.String()

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
	postResult.Uri = "http://localhost:4000/acme/new-authorization"
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

	postResult = true
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
	}

	return cR
}