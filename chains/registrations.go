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


func NewRegistrationTestChain() (ChainResult) {
	cR := ChainResult{Name: "new registration"}
	chainStart := time.Now()

	// generate a random phone number
	var buff bytes.Buffer
	buff.WriteString("+")
	mrand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		buff.WriteString(fmt.Sprintf("%d", mrand.Intn(9)))
	}
	regNum := buff.String()

	// create the registration object
	reg := fmt.Sprintf("{\"contact\":[\"tel:%s\"]}", regNum)

	// build the JWS object
	alg := jose.RSAPKCS1WithSHA256
	payload := []byte(reg)
	jws, err := jose.Sign(alg, TheKey, payload)

	// into JSON
	requestPayload, _ := json.Marshal(jws)
	// send a timed POST request
	client := &http.Client{}
	body, status, _, timing, err := timedPOST(client, "http://localhost:4000/acme/new-reg", requestPayload)
	var postResult requestResult
	postResult.Uri = "http://localhost:4000/acme/new-reg"
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

	var respReg core.Registration
	err = json.Unmarshal(body, &respReg)
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

	cR.Successful = true
	cR.Took = time.Since(chainStart)

	fmt.Println(respReg)

	return cR
}