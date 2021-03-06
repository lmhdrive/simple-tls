//     Copyright (C) 2020, IrineSistiana
//
//     This file is part of simple-tls.
//
//     mosdns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mosdns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	mathRand "math/rand"
)

func doServer(l net.Listener, tlsConfig *tls.Config, dst string, timeout time.Duration) error {

	for {
		clientRawConn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("l.Accept(): %w", err)
		}

		go func() {
			defer clientRawConn.Close()
			clientTLSConn := tls.Server(clientRawConn, tlsConfig)
			// check client conn before dial dst
			if err := clientTLSConn.Handshake(); err != nil {
				log.Printf("doServer: %s, clientTLSConn.Handshake: %v", clientRawConn.RemoteAddr(), err)
				return
			}

			dstConn, err := net.Dial("tcp", dst)
			if err != nil {
				log.Printf("doServer: %s: net.Dial: %v", clientRawConn.RemoteAddr(), err)
				return
			}
			defer dstConn.Close()

			if err := openTunnel(dstConn, clientTLSConn, timeout); err != nil {
				log.Printf("doServer: %s: openTunnel: %v", clientRawConn.RemoteAddr(), err)
			}
		}()
	}
}

func generateCertificate(serverName string) (dnsName string, keyPEM, certPEM []byte, err error) {
	//priv key
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return
	}

	//serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		err = fmt.Errorf("generate serial number: %v", err)
		return
	}

	// set DNSNames
	if len(serverName) == 0 {
		dnsName = randServerName()
	} else {
		dnsName = serverName
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: dnsName},
		DNSNames:     []string{dnsName},

		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return
	}
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return dnsName, keyPEM, certPEM, nil
}

func randServerName() string {
	return fmt.Sprintf("%s.%s", randStr(mathRand.Intn(5)+3), randStr(mathRand.Intn(3)+1))
}

func randStr(length int) string {
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	set := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}
