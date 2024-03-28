#!/bin/ash
FILE_CERT_NAME=servercert
openssl req -newkey rsa:4096 \
            -x509 \
            -sha256 \
            -days 365 \
            -nodes \
            -out "./../cmd/shortener/$FILE_CERT_NAME.crt" \
            -keyout "./../cmd/shortener/$FILE_CERT_NAME.key"