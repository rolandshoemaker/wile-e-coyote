package chains

import  (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/letsencrypt/boulder/core"
)

type requestResult struct {
    Uri        string        `json:"uri,omitempty"`        // endpoint being hit e.g. "/acme/new-reg"
    Took       time.Duration `json:"took,omitempty"`       // time request actually took
    Successful bool          `json:"successful,omitempty"` // if the *right* thing happened (subjective...)
    Error      string        `json:"error,omitempty"`      // if not right thing, what went wrong
}

type ChainResult struct {
    Name              string          `json:"chainname,omitempty"`         // which chain is this result from
    Successful        bool            `json:"chainsuccessful,omitempty"`   // was the entire chain successful?
    Took              time.Duration   `json:"chaintook,omitempty"`         // how long did the entire chain take
    IndividualResults []requestResult `json:"individualresults,omitempty"` // individual requestResults
}

type boulderReg struct {
	Reg    core.Registration
	Key interface{}
}

type boulderAuth struct {
	Auth    core.Authorization
	// something to tie to a reg?
}

type boulderCert struct {
	Cert    core.Certificate
	// something to tie to a reg?
}

type ChainContext struct {
	Registrations  []boulderReg
	Authorizations []string
	Certificates   []string
}

type testChain struct {
	TestFunc     func(ChainContext) (ChainResult, ChainContext)
	Requirements []string
}

var TestChains []testChain = []testChain{
	testChain{TestFunc: NewRegistrationTestChain, Requirements: []string{}},
	// testChain{TestFunc: NewAuthorizationTestChain, Requirements: []string{"registration"}},
	// testChain{TestFunc: NewCertificateTestChain, Requirements: []string{"registration", "authorization"}},
	// testChain{TestFunc: RevokeCertificateTestChain, Requirements: []string{"registration", "certificate"}},
}

// public functions

// return a *random* (ish) test chain from TestChains that
// satifies testChain.requirements (based on whats in SQL)
func GetChain() (func(ChainContext) (ChainResult, ChainContext), ChainContext) {
	var randChain testChain
	var cC ChainContext
	var satisfied bool
	for {
		randChain = TestChains[rand.Intn(len(TestChains))]
		satisfied, cC = getRequirements(randChain.Requirements)
		if satisfied {
			break
		}
	}
	return randChain.TestFunc, cC
}

func UpdateContext(oldContext, newContext ChainContext) {
	// figure out whats been added / removed (map approach?)
	// and update SQL to reflect that
}

func SendStats(result ChainResult) {

}

// internal stuff

func getRequirements(reqs []string) (satisfied bool, cC ChainContext) {
	for _, req := range reqs {
		switch req {
		case "registration":

		case "authorization":

		case "certificate":

		}
	}

	satisfied = true
	return
}

// utility functions

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

func timedGET(url string) {

}


