package chains

import  (
	"github.com/garyburd/redigo/redis"
)

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

