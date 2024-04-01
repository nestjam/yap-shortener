package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

func Generate() (certPEM bytes.Buffer, privateKeyPEM bytes.Buffer, err error) {
	const (
		op           = "generate"
		serialNumber = 1658
		years        = 1
		localhost    = "127.0.0.1"
	)

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Organization: []string{"None"},
			Country:      []string{"RU"},
		},
		IPAddresses: []net.IP{
			net.ParseIP(localhost),
			net.IPv6loopback,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(years, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature,
	}

	const bits = 4096
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		err = errors.Wrap(err, op)
		return
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		err = errors.Wrap(err, op)
		return
	}

	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		err = errors.Wrap(err, op)
		return
	}

	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		err = errors.Wrap(err, op)
		return
	}

	return
}
