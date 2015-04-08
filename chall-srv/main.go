package main

import (
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"crypto/tls"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/core"
	"github.com/garyburd/redigo/redis"
)

func newPool(server, password string) *redis.Pool {
    return &redis.Pool{
        MaxIdle: 3,
        IdleTimeout: 240 * time.Second,
        Dial: func () (redis.Conn, error) {
            c, err := redis.Dial("tcp", server)
            if err != nil {
                return nil, err
            }
            if password != "" {
	            if _, err := c.Do("AUTH", password); err != nil {
	                c.Close()
	                return nil, err
	            }
            }
            return c, err
        },
        TestOnBorrow: func(c redis.Conn, t time.Time) error {
            _, err := c.Do("PING")
            return err
        },
    }
}

var (
    pool *redis.Pool
)

func genSelfSigned(dnsNames []string, pubKey *rsa.PublicKey) (cert *tls.Certificate) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1337),
		Subject: pkix.Name{
			Organization: []string{"wile e coyote llc"},
		},
		NotBefore: time.Now(),
		NotAfter: time.Now().AddDate(0, 0, 1),

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: dnsNames,
	}
	
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("couldn't generate rsa 2048 key", err)
		return &tls.Certificate{}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, pubKey, priv)
	if err != nil {
		fmt.Println("couldn't create cert", err)
		return &tls.Certificate{}
	}

	cert = &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey: priv,
	}
	return
}

func getPubKey(dnsName string, rConn redis.Conn) (pubKey *rsa.PublicKey, err error) {
	rKey := fmt.Sprintf("pk:%s", dnsName)
	var pubBytes []byte
	pubBytes, err = redis.Bytes(rConn.Do("GET", rKey))
	if err != nil {
		return
	}

	var block *pem.Block
	block, _ = pem.Decode(pubBytes)

	var pub interface{}
	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		err = fmt.Errorf("Didn't get a RSA public key...")
		return
	}

	return
}

// creates the right certificate with relevant DNS names and public key based on provided ServerName
// (from SNI).
func getCert(clientHello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	rConn := pool.Get()
	defer rConn.Close()

	var dnsNames []string
	//var publicKey string
	if strings.HasSuffix(clientHello.ServerName, ".acme.invalid") {
		// DVSNI challenge so lets compute Z... FUN
		nonce := clientHello.ServerName[0:len(clientHello.ServerName)-13]

		var packed string
		// this may conflict since the draft implicitly says nonces may be reused
		// should CHECK the boulder src to see what it does...
		packed, err = redis.String(rConn.Do("GET", nonce))
		if err != nil {
			return
		}

		//assume packed is in form 'dnsName:r:s:pubKey'
		unpacked := strings.SplitN(packed, ":", 4)
		if len(unpacked) < 4 {
			err = fmt.Errorf("Value for key [%s] in incorrect format [%s]", nonce, packed)
			return
		}

		var R, S []byte
		R, err = core.B64dec(unpacked[1])
		if err != nil {
			return
		}
		S, err = core.B64dec(unpacked[2])
		if err != nil {
			return
		}
		RS := append(R, S...)
		z := sha256.Sum256(RS)
		zName := hex.EncodeToString(z[:])
		zName = fmt.Sprintf("%s.acme.invalid", zName)

		domain := unpacked[0]
		if err != nil {
			return
		}
		dnsNames = append(dnsNames, domain, zName)
	} else {
		dnsNames = []string{clientHello.ServerName}
	}
	
	var pubKey *rsa.PublicKey
	pubKey, err = getPubKey(dnsNames[0], rConn)
	if err != nil {
		return
	}

	cert = genSelfSigned(dnsNames, pubKey)
	return
}

func listenAndMeepMeep(srv *http.Server) error {
	nullPk := rsa.PublicKey{N:big.NewInt(1337) , E: 10}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*genSelfSigned([]string{"roadrunner"}, &nullPk)}, // because Listener requires at least one
		ClientAuth: tls.NoClientCert,                                               // and it'll never actually be served
		GetCertificate: getCert,
		NextProtos: []string{"http/1.1"},
	}

	conn, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		fmt.Println("couldn't listen", err)
	}

	tlsListener := tls.NewListener(conn, tlsConfig)
	return srv.Serve(tlsListener)
}

func handler(w http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.TLS.ServerName, ".acme.invalid") {
		rConn := pool.Get()
		defer rConn.Close()

		key := fmt.Sprintf("ct:%s", req.TLS.ServerName)
		token, err := redis.Bytes(rConn.Do("GET", key))
		if err != nil {
			fmt.Println("couldn't get challenge token", err)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write(token)
	}
}

func main() {
	pool = newPool(":6379", "")

	http.HandleFunc("/", handler)

	httpsServer := &http.Server{Addr: ":443"}
	listenAndMeepMeep(httpsServer)
}
