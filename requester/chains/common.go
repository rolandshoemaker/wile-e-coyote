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

type testChain struct {
	TestFunc     func() ChainResult
	Requirements []string
}

var TestChains []testChain = []testChain{
	TestChain{testFunc: NewRegistrationTestChain, Requirements: []string{}},
	TestChain{testFunc: NewAuthorizationTestChain, Requirements: []string{"registration"}},
	TestChain{testFunc: NewCertificateTestChain, Requirements: []string{"registration", "authorization"}},
	TestChain{testFunc: RevokeCertificateTestChain, Requirements: []string{"registration", "authorization", "certificate"}},
}

// public functions

// return a *random* (ish) test chain from TestChains that
// satifies testChain.requirements (based on whats in SQL)
func GetChain() func() ChainResult {
	var randChain testChain
	for {
		randChain = TestChains[rand.Intn(len(TestChains)-1)]
		if satisfiesRequirements(randChain) {
			break
		}
	}
	return randChain.TestFunc
}

// internal stuff

func satisfiesRequirements(c ChainResult) bool {
	satisfied := true
	for _, req := range c.Requirements {
		if !satisfied {
			break
		}
		switch req {
		case "registration":

		case "authorization":

		case "certificate":

		}
	}

	return satisfied
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

func timedPOST(url string, data interface{}) {

}

func timedGET(url string) {

}


