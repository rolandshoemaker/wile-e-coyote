package chains

import (
	"bytes"
	"encoding/json"
	mrand "math/rand"
	// "crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/letsencrypt/boulder/core"	
	"github.com/letsencrypt/boulder/jose"	
)

func bigIntFromB64(b64 string) *big.Int {
	bytes, _ := jose.B64dec(b64)
	x := big.NewInt(0)
	x.SetBytes(bytes)
	return x
}

func intFromB64(b64 string) int {
	return int(bigIntFromB64(b64).Int64())
}

var n *big.Int = bigIntFromB64("n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw")
var e int = intFromB64("AQAB")
var d *big.Int = bigIntFromB64("bWUC9B-EFRIo8kpGfh0ZuyGPvMNKvYWNtB_ikiH9k20eT-O1q_I78eiZkpXxXQ0UTEs2LsNRS-8uJbvQ-A1irkwMSMkK1J3XTGgdrhCku9gRldY7sNA_AKZGh-Q661_42rINLRCe8W-nZ34ui_qOfkLnK9QWDDqpaIsA-bMwWWSDFu2MUBYwkHTMEzLYGqOe04noqeq1hExBTHBOBdkMXiuFhUq1BU6l-DqEiWxqg82sXt2h-LMnT3046AOYJoRioz75tSUQfGCshWTBnP5uDjd18kKhyv07lhfSJdrPdM5Plyl21hsFf4L_mHCuoFau7gdsPfHPxxjVOcOpBrQzwQ")
var p *big.Int = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc")
var q *big.Int = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc")

var theKey rsa.PrivateKey = rsa.PrivateKey{
	PublicKey: rsa.PublicKey{N: n, E: e},
	D:         d,
	Primes:    []*big.Int{p, q},
}


func NewRegistrationTestChain(cC ChainContext) (ChainResult, ChainContext) {
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
	// priv, err := rsa.GenerateKey(rand.Reader, 2048)
	payload := []byte(reg)
	jws, err := jose.Sign(alg, theKey, payload)

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
		return cR, cC
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
		return cR, cC
	}

	postResult.Successful = true
	cR.IndividualResults = append(cR.IndividualResults, postResult)	

	var newReg boulderReg
	newReg.Reg = respReg
	newReg.Key = theKey

	cR.Successful = true
	cR.Took = time.Since(chainStart)

	cC.Registrations = append(cC.Registrations, newReg)

	fmt.Println(respReg)

	return cR, cC
}