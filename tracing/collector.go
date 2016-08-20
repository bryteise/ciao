//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package trace

import (
	"fmt"
	"sync"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

// TracePort is the default ciao trace collector SSNTP port.
const TracePort = 9888

type spanCache struct {
	spans []payloads.Span
	lock  sync.RWMutex
}

// CollectorConfig represents a collector configuration.
type CollectorConfig struct {
	// Store is the storage backend driver for storing spans.
	Store SpanStore

	// Port is the SSNTP port where the collector will listen
	// for spans.
	Port int

	// CACert is the Certification Authority certificate path
	// to use when verifiying the peer identity.
	CAcert string

	// Cert is the tracer x509 signed certificate path.
	Cert string
}

// Collector represents a ciao trace collector.
// Ciao trace collectors listen on dedicated SSNTP ports for
// out-of-band trace span arrays payloads to collect and store
// them. They are also responsible for responding to trace
// queries.
type Collector struct {
	cache spanCache
	store SpanStore

	port   int
	caCert string
	cert   string
	ssntp  ssntp.Server
}

// NewCollector creates a new trace collector.
func NewCollector(config *CollectorConfig) (*Collector, error) {
	if config.Store == nil {
		config.Store = &Noop{}
	}

	if config.Port == 0 {
		config.Port = TracePort
	}

	if config.CAcert == "" {
		return nil, fmt.Errorf("Missing CA")
	}

	if config.Cert == "" {
		return nil, fmt.Errorf("Missing private key")
	}

	collector := &Collector{
		store:  config.Store,
		port:   config.Port,
		caCert: config.CAcert,
		cert:   config.Cert,
	}

	return collector, nil
}
