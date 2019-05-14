package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/op/go-logging"
)

var (
	log, _      = logging.GetLogger("encrypt53")
	logLevel, _ = logging.LogLevel(os.Getenv("LOG_LEVEL"))

	acmeUrl              = os.Getenv("ACME_URL")
	contact              = os.Getenv("CONTACT")
	notificationTopicArn = os.Getenv("NOTIFICATION_TOPIC_ARN")
	renewBefore          = time.Now().Round(time.Hour).Add(time.Hour * 24 * time.Duration(parseIntFromEnv("RENEW_BEFORE", 14)))
	runLocal             = os.Getenv("RUN_LOCAL")
	storeBucket          = os.Getenv("BUCKET_NAME")

	acmeClient *AcmeClient
	notifyer   *Notifyer
)

type Event struct {
	Fqdn string `json:"fqdn"`
}

func main() {
	initContext(context.Background())

	if runLocal == "" {
		lambda.Start(HandleRequest)
	} else if err := HandleRequest(context.Background(), Event{Fqdn: os.Getenv("FQDN")}); err != nil {
		log.Fatalf("error running lambda locally: %s", err.Error())
	}
}

func HandleRequest(ctx context.Context, event Event) error {
	if event.Fqdn == "" {
		return notifyer.Error("Error renewing certificates",
			acmeClient.RenewAllDueCertificates(ctx, renewBefore))
	} else {
		return notifyer.Error(fmt.Sprintf("Error creating certificate for %s", event.Fqdn),
			acmeClient.CreateCertificate(ctx, event.Fqdn))
	}
}

func initContext(ctx context.Context) {
	logging.SetLevel(logLevel, "")

	sess := session.Must(session.NewSession())
	dns := NewDnsClient(sess)
	store := NewCertStore(sess, storeBucket)

	var err error
	if acmeClient, err = NewAcmeClient(ctx, dns, store, contact, acmeUrl); err != nil {
		log.Fatalf("error creating acme client: %s", err.Error())
	}

	notifyer = NewNotifyer(sess, notificationTopicArn)
}

func parseIntFromEnv(key string, def int64) int64 {
	if i, err := strconv.ParseInt(os.Getenv(key), 10, 64); err != nil {
		return def
	} else {
		return i
	}
}
