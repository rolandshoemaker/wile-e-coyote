package chains

import (
	"bytes"
	"math/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/letsencrypt/boulder/jose"	
)

func NewRegistrationTestChain(chainContext) (ChainResult, chainContext) {
	cR := ChainResult{ChainName: "new registration"}

	var buff bytes.Buffer
	buff.WriteString("+")
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		buff.WriteString(fmt.Sprintf("%d", rand.Intn(9)))
	}
	regNum := buff.String()

	reg := fmt.Sprintf("{\"contact\":[\"tel:%s\"]}", regNum)

	alg := jose.RSAPKCS1WithSHA256
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	paylod := []byte(reg)

	jws, _ := jose.Sign(alg, priv, payload)

	requestPayload, _ := json.Marshal(jws)

	client := &http.Client{}

	body, status, headers, timing := timedPOST(client, "http://localhost:4000/acme/new-reg", requestPayload)

	return cR, chainContext
}