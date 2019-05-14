# encrypt53

This tool creates let's encrypt certificates for your endpoints.
It runs DNS-01 validation with AWS route53 to claim ownership.
Certificates are stored in AWS S3.

Encrypt53 is designed to run as lambda every now and then supporting renewal of certificates.

## Configuration

Encrypt53 is configurable via the following environment variables:

* `ACME_URL`: Optional ACME URL, defaults to let's encrypt live system
* `BUCKET_NAME`: AWS S3 bucket name for storing registered certificates and keys
* `CONTACT`: Your email address used for registration at let's encrypt
* `LOG_LEVEL`: Log level, one of `CRITICAL`, `ERROR`, `WARNING`, `NOTICE`, `INFO` or `DEBUG`
* `NOTIFICATION_TOPIC_ARN`: Optional AWS SNS topic arn. Encrypt will send error notifications to this topic if set.
* `RENEW_BEFORE`:  Optional number of days before end of valid period a certificate is renewed. Defaults to 14 days.
* `RUN_LOCAL`: Optional toggle to run encrypt53 locally instead as AWS lambda. Leave empty to run as lambda.

## Usage

There are two modes:

* Event triggered to create a new certificate for a given FQDN.
  Supported event structure is `{"fqdn': "whatever.example.com"}`
* If no event or any other event is given, all all due certificates are renewed.
  E.g. triggered by cloudwatch events once a day.

## Persistence

Keys and certificates are stored in your AWS S3 bucket under following keys:

* `/keys/<fqdn>.key` as x509 asn.1 DER format
* `/certs/<fqdn>.crt` as x509 certificate chain PEM format
