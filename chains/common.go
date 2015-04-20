// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//

package chains

import  (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/cactus/go-statsd-client/statsd"

	"github.com/letsencrypt/boulder/jose"
)

type requestResult struct {
    Uri        string        // endpoint being hit e.g. "/acme/new-reg"
    Took       time.Duration // time request actually took
    Successful bool          // if the *right* thing happened (subjective...)
    Error      string        // if not right thing, what went wrong
}

type ChainResult struct {
    Name              string          // which chain is this result from
    Successful        bool            // was the entire chain successful?
    Took              time.Duration   // how long did the entire chain take
    IndividualResults []requestResult // individual requestResults
}

type testChain struct {
	TestFunc     func() (ChainResult)
}

var TestChains []testChain = []testChain{
	testChain{TestFunc: NewRegistrationTestChain},
	testChain{TestFunc: NewAuthorizationTestChain},
	// testChain{TestFunc: NewCertificateTestChain},
	// testChain{TestFunc: RevokeCertificateTestChain},
}

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

var TheKey rsa.PrivateKey = rsa.PrivateKey{
	PublicKey: rsa.PublicKey{N: n, E: e},
	D:         d,
	Primes:    []*big.Int{p, q},
}

var SimpleHTTPSChalls map[string]string = make(map[string]string)
type DvsniChall struct {
	Domain string
	Z      string
}
var DvsniChalls map[string]DvsniChall = make(map[string]DvsniChall)

// public functions

// return a *random* (ish) test chain from TestChains that
// satifies testChain.requirements (based on whats in SQL)
func GetChain() (func() (ChainResult)) {
	var randChain testChain
	randChain = TestChains[rand.Intn(len(TestChains))]

	return randChain.TestFunc
}

func SendStats(stats statsd.Statter, result ChainResult) {

}

// utility functions

// borrow stuff from boulder SA
func existingRegistrations() bool {
	return true
}

func getAuthorization() {
	// return an authorization from sql or nil?
	// query directly from boulder sql!
}

func getCertificate() {
	// return certificate from sql or nil?
	// query directly from boulder sql!
}

func setRedisPubKey(dnsName string, pubKey *rsa.PublicKey, rConn redis.Conn) error {
	rKey := fmt.Sprintf("pk:%s", dnsName)
	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)

	ok, err := redis.Bool(rConn.Do("SET", rKey, pubBytes))
	if err != nil {
		return err
	}
	if !ok {
		return err
	}

	return nil
}

func timedPOST(client *http.Client, url string, data []byte) (body []byte, status int, headers http.Header, since time.Duration, err error) {
	sTime := time.Now()
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		since = time.Since(sTime)
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		since = time.Since(sTime)
		return
	}
	since = time.Since(sTime)
	status = resp.StatusCode
	headers = resp.Header
	return
}

func timedGET(client *http.Client, url string) (body []byte, status int, headers http.Header, since time.Duration, err error) {
	sTime := time.Now()
	req, _ := http.NewRequest("GET", url, nil)

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		since = time.Since(sTime)
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		since = time.Since(sTime)
		return
	}
	since = time.Since(sTime)
	status = resp.StatusCode
	headers = resp.Header
	return
}


