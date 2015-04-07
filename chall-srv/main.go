package main

import (
	"fmt"
	"net"
	"crypto/tls"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"time"
)

func genSingleSelfSigned(sni []string) (cert *tls.Certificate) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1337),
		Subject: pkix.Name{
			Organization: []string{"wile e coyote llc"},
		},
		NotBefore: time.Now(),
		NotAfter: time.Now().AddDate(0, 0, 15),

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: sni,
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
	// should also get public key for the realy domain (+with a else here!)

	cert = genSingleSelfSigned(dnsNames)
	return
}

func main() {
	ln, err := net.Listen("tcp", ":443")
	if err != nil {
		fmt.Println("couldn't listen wut", err)
		return
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println("couldn't accept new connection", err)
			return
		}
		defer c.Close()
		go func(c net.Conn) {
			tlsConfig := tls.Config{
				Certificates: []tls.Certificate{*genSingleSelfSigned([]string{"null"})}, // because servers require at least one...
				ClientAuth: tls.NoClientCert,
				GetCertificate: getCert,
			}
			tlsServer := tls.Server(c, &tlsConfig)

			err = tlsServer.Handshake()
			if err != nil {
				fmt.Println("hm", err)
				return
			}

			var stuff []byte
			_, err = tlsServer.Read(stuff)
			if err != nil {
				fmt.Println("couldnt read!", err)
				return
			}
			fmt.Println(string(stuff))

			tlsServer.Write([]byte("derpderp"))

			tlsServer.Close()
			return
		}(c)
	}
}
