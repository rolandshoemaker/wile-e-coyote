package chains

import  (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"math/rand"
	"time"

	"github.com/garyburd/redigo/redis"
)

type requestResult struct {
    Uri        string        `json:"uri,omitempty"`        // endpoint being hit e.g. "/acme/new-reg"
    Took       time.Duration `json:"took,omitempty"`       // time request actually took
    Successful bool          `json:"successful,omitempty"` // if the *right* thing happened (subjective...)
    Error      string        `json:"error,omitempty"`      // if not right thing, what went wrong
}

type ChainResult struct {
    ChainName         string          `json:"chainname,omitempty"`         // which chain is this result from
    ChainSuccessful   bool            `json:"chainsuccessful,omitempty"`   // was the entire chain successful?
    ChainTook         time.Duration   `json:"chaintook,omitempty"`         // how long did the entire chain take
    IndividualResults []requestResult `json:"individualresults,omitempty"` // individual requestResults
}

type chainContext struct {
	Registrations  []string
	Authorizations []string
	Certificates   []string
}

type testChain struct {
	TestFunc     func(chainContext) (ChainResult, chainContext)
	Requirements []string
}

var TestChains []testChain = []testChain{
	testChain{TestFunc: NewRegistrationTestChain, Requirements: []string{}},
	testChain{TestFunc: NewAuthorizationTestChain, Requirements: []string{"registration"}},
	testChain{TestFunc: NewCertificateTestChain, Requirements: []string{"registration", "authorization"}},
	testChain{TestFunc: RevokeCertificateTestChain, Requirements: []string{"registration", "certificate"}},
}

// public functions

// return a *random* (ish) test chain from TestChains that
// satifies testChain.requirements (based on whats in SQL)
func GetChain() func(chainContext) (ChainResult, chainContext) {
	var randChain testChain
	for {
		randChain = TestChains[rand.Intn(len(TestChains)-1)]
		cC := getRequirements(randChain.Requirements)
		if cC != nil {
			break
		}
	}
	return randChain.TestFunc, cC
}

func UpdateContext(oldContext, newContext chainContext) {
	// figure out whats been added / removed (map approach?)
	// and update SQL to reflect that
	if oldContext.Requirements != newContext.Requirements {

	}
	if oldContext.Authorizations != newContext.Authorizations {

	}
	if oldContext.Certificates != newContext.Certificates {

	}
}

func SendStats(result ChainResult) {

}

// internal stuff

func getRequirements(reqs []string) (cC chainContext) {
	for _, req := range reqs {
		switch req {
		case "registration":

		case "authorization":

		case "certificate":

		}
	}

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

func timedPOST(client, url string, data []byte]) {
	sTime := time.Now()
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {

	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return body, resp.Status, resp.Header, time.Since(sTime)
}

func timedGET(url string) {

}


