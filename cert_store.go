package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const (
	notAfterKey = "Not-After"
)

type CertStore struct {
	client s3iface.S3API
	bucket string
}

func NewCertStore(sess *session.Session, bucketName string) *CertStore {
	return &CertStore{
		client: s3.New(sess),
		bucket: bucketName,
	}
}

func (cs *CertStore) GetKey(fqdn string) (crypto.Signer, error) {
	path := keyPath(fqdn)

	if key, err := cs.getKey(path); err == nil {
		return key, nil
	}

	log.Infof("generating new private key for %s", fqdn)
	if key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
		return nil, err
	} else {
		return key, cs.putKey(path, key)
	}
}

func (cs *CertStore) getKey(path string) (*ecdsa.PrivateKey, error) {
	if result, err := cs.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(cs.bucket),
		Key:    aws.String(path),
	}); err != nil {
		return nil, err
	} else if der, err := ioutil.ReadAll(result.Body); err != nil {
		return nil, err
	} else {
		return x509.ParseECPrivateKey(der)
	}
}

func (cs *CertStore) putKey(path string, key *ecdsa.PrivateKey) error {
	if der, err := x509.MarshalECPrivateKey(key); err != nil {
		return err
	} else if _, err := cs.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(cs.bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(der),
	}); err != nil {
		return err
	} else {
		return nil
	}
}

func (cs *CertStore) PutCert(fqdn string, chain *tls.Certificate) error {
	var buf bytes.Buffer
	for _, cert := range chain.Certificate {
		block := &pem.Block{Type: "CERTIFICATE", Bytes: cert}
		if err := pem.Encode(&buf, block); err != nil {
			return err
		}
	}

	_, err := cs.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(cs.bucket),
		Key:    aws.String(certPath(fqdn)),
		Body:   bytes.NewReader(buf.Bytes()),
		Metadata: map[string]*string{
			notAfterKey: aws.String(chain.Leaf.NotAfter.Format(time.RFC3339)),
		},
	})
	return err
}

func (cs *CertStore) ListCerts() ([]string, error) {
	certs := make([]string, 0)

	if err := cs.client.ListObjectsPages(
		&s3.ListObjectsInput{
			Bucket: aws.String(cs.bucket),
			Prefix: aws.String("certs/"),
		},
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, o := range page.Contents {
				certs = append(certs, path2fqdn(*o.Key))
			}
			return false
		}); err != nil {
		return nil, err
	}
	return certs, nil
}

func (cs *CertStore) ValidUntil(fqdn string) (time.Time, error) {
	if result, err := cs.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(cs.bucket),
		Key:    aws.String(certPath(fqdn)),
	}); err != nil {
		return time.Time{}, err
	} else {
		return time.Parse(time.RFC3339, *result.Metadata[notAfterKey])
	}
}

func keyPath(fqdn string) string {
	return fmt.Sprintf("keys/%s.key", fqdn)
}

func certPath(fqdn string) string {
	return fmt.Sprintf("certs/%s.crt", fqdn)
}

func path2fqdn(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, "certs/"), ".crt")
}
