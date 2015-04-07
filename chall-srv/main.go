package main

import (
	"fmt"
	"net"
	"net/http"
	"crypto/tls"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"time"
)

func genSelfSigned(dnsNames []string) (cert *tls.Certificate) {
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
	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
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

// creates the right certificate with relevant DNS names based on provided ServerName
// (from SNI).
func getCert(clientHello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	dnsNames := []string{clientHello.ServerName}
	if strings.HasSuffix(clientHello.ServerName, ".acme.invalid") {
		// actually check what this is by getting secret from servername
		domain := "example.com"
		dnsNames = append(dnsNames, domain)
	}
	// should also get public key for the real domain (+with a else here! or something)

	cert = genSelfSigned(dnsNames)
	return
}

func listenAndMeepMeep(srv *http.Server) error {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*genSelfSigned([]string{"null"})}, // because servers require at least one...
		ClientAuth: tls.NoClientCert,
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
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("This is an example server.\n"))
}

func main() {
	http.HandleFunc("/", handler)

	httpsServer := &http.Server{Addr: ":443"}
	listenAndMeepMeep(httpsServer)
}
