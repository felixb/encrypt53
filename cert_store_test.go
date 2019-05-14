package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/felixb/encrypt53/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	cert_store_test_bucket = "some-bucket"
)

func TestCertStore_GetKey_newKey(t *testing.T) {
	client, s := givenS3Client()

	client.On("GetObject", mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == cert_store_test_bucket &&
			*input.Key == "keys/server.some.domain.key"
	})).Return(nil, errors.New("not found"))
	client.On("PutObject", mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == cert_store_test_bucket &&
			*input.Key == "keys/server.some.domain.key"
	})).Return(nil, nil)

	key, err := s.GetKey("server.some.domain")

	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.IsType(t, &ecdsa.PrivateKey{}, key)
	client.AssertExpectations(t)
}

func TestCertStore_GetKey_existingKey(t *testing.T) {
	client, s := givenS3Client()

	expectedKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NotNil(t, expectedKey)
	der, _ := x509.MarshalECPrivateKey(expectedKey)
	assert.NotNil(t, der)

	client.On("GetObject", mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == cert_store_test_bucket &&
			*input.Key == "keys/server.some.domain.key"
	})).Return(&s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader(der)),
	}, nil)

	key, err := s.GetKey("server.some.domain")

	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, expectedKey, key)
	client.AssertExpectations(t)
}

func TestCertStore_ValidUntil(t *testing.T) {
	client, s := givenS3Client()
	someTime := time.Now().Round(time.Second)

	client.On("HeadObject", mock.MatchedBy(func(input *s3.HeadObjectInput) bool {
		return *input.Bucket == cert_store_test_bucket &&
			*input.Key == "certs/server.some.domain.crt"
	})).Return(&s3.HeadObjectOutput{
		Metadata: map[string]*string{"Not-After": aws.String(someTime.Format(time.RFC3339))},
	}, nil)

	vu, err := s.ValidUntil("server.some.domain")

	assert.NoError(t, err)
	assert.Equal(t, someTime, vu)
	client.AssertExpectations(t)
}

func givenS3Client() (*mocks.S3API, *CertStore) {
	client := new(mocks.S3API)
	store := &CertStore{
		client: client,
		bucket: cert_store_test_bucket,
	}
	return client, store
}
