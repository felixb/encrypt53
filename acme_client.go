package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/acme"
)

type AcmeClient struct {
	client *acme.Client
	dns    *DnsClient
	store  *CertStore
}

func NewAcmeClient(ctx context.Context, dns *DnsClient, store *CertStore, contact string, apiUrl string) (*AcmeClient, error) {
	var url string
	if apiUrl == "" {
		url = acme.LetsEncryptURL
	} else {
		url = apiUrl
	}
	ac := &AcmeClient{
		client: &acme.Client{DirectoryURL: url},
		store:  store,
		dns:    dns,
	}

	if key, err := store.GetKey(contact); err != nil {
		return nil, err
	} else {
		ac.client.Key = key
	}
	account := &acme.Account{Contact: []string{fmt.Sprintf("mailto:%s", contact)}}
	account, err := ac.client.Register(ctx, account, acme.AcceptTOS)
	if ae, ok := err.(*acme.Error); err == nil || ok && ae.StatusCode == http.StatusConflict {
		log.Debugf("ac already registered")
	} else {
		return nil, err
	}

	return ac, nil
}

func (ac *AcmeClient) CreateCertificate(ctx context.Context, fqdn string) error {
	if err := ac.authorizeDomain(ctx, fqdn); err != nil {
		return err
	} else if key, err := ac.store.GetKey(fqdn); err != nil {
		return err
	} else if csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: fqdn},
	}, key); err != nil {
		return err
	} else if der, _, err := ac.client.CreateCert(ctx, csr, 0, true); err != nil {
		return err
	} else if len(der) == 0 {
		return errors.New("API returned zero length certificate chain")
	} else if leaf, err := x509.ParseCertificate(der[0]); err != nil {
		return err
	} else {
		chain := &tls.Certificate{
			Certificate: der,
			Leaf:        leaf,
		}
		return ac.store.PutCert(fqdn, chain)
	}
}

func (ac *AcmeClient) RenewAllDueCertificates(ctx context.Context, notAfterLaterAs time.Time) error {
	if fqdns, err := ac.store.ListCerts(); err != nil {
		return err
	} else {
		dueFqdns := make([]string, 0)
		for _, fqdn := range fqdns {
			if validUntil, err := ac.store.ValidUntil(fqdn); err != nil {
				log.Errorf("error fetching due date for %s: %s", fqdn, err.Error())
			} else if validUntil.Before(notAfterLaterAs) {
				dueFqdns = append(dueFqdns, fqdn)
			}
		}

		if len(dueFqdns) == 0 {
			log.Infof("all of %d certificates are up to date", len(fqdns))
			return nil
		} else {
			log.Infof("%d of %d certificates need refreshing", len(dueFqdns), len(fqdns))
			return ac.createCertificates(ctx, dueFqdns)
		}
	}
}

func (ac *AcmeClient) createCertificates(ctx context.Context, fqdns []string) error {
	errors := 0
	for _, fqdn := range fqdns {
		if err := ac.CreateCertificate(ctx, fqdn); err != nil {
			errors += 1
		}
	}

	if errors > 0 {
		return fmt.Errorf("%d errors occured during creation of %d certificates", errors, len(fqdns))
	} else {
		return nil
	}
}

func (ac *AcmeClient) authorizeDomain(ctx context.Context, fqdn string) error {
	log.Infof("validating domain %s", fqdn)
	if authz, err := ac.client.Authorize(ctx, fqdn); err != nil {
		return err
	} else if authz.Status == acme.StatusValid {
		log.Debugf("domain %s is already validated", fqdn)
		return nil
	} else if authz.Status == acme.StatusInvalid {
		return fmt.Errorf("invalid authorization: %s", authz.URI)
	} else {
		challenge := dnsChallenge(authz.Challenges)
		if value, err := ac.client.DNS01ChallengeRecord(challenge.Token); err != nil {
			return err
		} else {
			//noinspection ALL
			//defer ac.dns.DeleteChallengeRecord(fqdn, value)
			if err := ac.dns.AddChallengeRecord(fqdn, value); err != nil {
				return err
			}

			if _, err := ac.client.Accept(ctx, challenge); err != nil {
				return err
			}

			if _, err := ac.client.WaitAuthorization(ctx, authz.URI); err == nil {
			} else {
				return err
			}
		}
	}

	return nil
}

func dnsChallenge(chal []*acme.Challenge) *acme.Challenge {
	for _, c := range chal {
		if c.Type == "dns-01" {
			return c
		}
	}
	return nil
}
