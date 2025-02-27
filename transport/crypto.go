// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !requirefips

package transport // import "go.elastic.co/apm/v2/transport"

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"go.elastic.co/apm/v2/internal/configutil"
)

const envVerifyServerCert = "ELASTIC_APM_VERIFY_SERVER_CERT"

func checkVerifyServerCert() (bool, error) {
	return configutil.ParseBoolEnv(envVerifyServerCert, true)
}

func addCertPath(tlsClientConfig *tls.Config) error {
	if serverCertPath := os.Getenv(envServerCert); serverCertPath != "" {
		serverCert, err := loadCertificate(serverCertPath)
		if err != nil {
			return errors.Wrapf(err, "failed to load certificate from %s", serverCertPath)
		}
		// Disable standard verification, we'll check that the
		// server supplies the exact certificate provided.
		tlsClientConfig.InsecureSkipVerify = true
		tlsClientConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return verifyPeerCertificate(rawCerts, serverCert)
		}
	}
	if serverCACertPath := os.Getenv(envServerCACert); serverCACertPath != "" {
		rootCAs := x509.NewCertPool()
		additionalCerts, err := os.ReadFile(serverCACertPath)
		if err != nil {
			return errors.Wrapf(err, "failed to load root CA file from %s", serverCACertPath)
		}
		if !rootCAs.AppendCertsFromPEM(additionalCerts) {
			return fmt.Errorf("failed to load CA certs from %s", serverCACertPath)
		}
		tlsClientConfig.RootCAs = rootCAs
	}

	return nil
}

func loadCertificate(path string) (*x509.Certificate, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for {
		var certBlock *pem.Block
		certBlock, pemBytes = pem.Decode(pemBytes)
		if certBlock == nil {
			return nil, errors.New("missing or invalid certificate")
		}
		if certBlock.Type == "CERTIFICATE" {
			return x509.ParseCertificate(certBlock.Bytes)
		}
	}
}

func verifyPeerCertificate(rawCerts [][]byte, trusted *x509.Certificate) error {
	if len(rawCerts) == 0 {
		return errors.New("missing leaf certificate")
	}
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return errors.Wrap(err, "failed to parse certificate from server")
	}
	if !cert.Equal(trusted) {
		return errors.New("failed to verify server certificate")
	}
	return nil
}
