package chains

import  (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
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
	testFunc     func() ChainResult
	requirements []string
}

var TestChains []testChain = []testChain{testChain{testFunc: func() ChainResult {return ChainResult{}}, requirements: []string{"woop"}}}

// public functions

// return a *random* test chain from TestChains that
// satifies testChain.requirements (based on whats in SQL)
func GetChain() func() ChainResult {
	return TestChains[0].testFunc
}

// internal stuff

func satifiesRequirements() bool {
	return true
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


