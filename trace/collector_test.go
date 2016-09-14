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
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/testutil"
)

func getTestSpans() (payloads.Spans, error) {
	ts, err := time.Parse(time.RFC3339, testutil.SpanTimeStamp)
	if err != nil {
		return payloads.Spans{}, err
	}
	spans := payloads.Spans{}
	span := payloads.Span{
		UUID:             testutil.SpanUUID,
		ParentUUID:       testutil.ParentSpanUUID,
		CreatorUUID:      testutil.AgentUUID,
		Component:        string(Anonymous),
		Timestamp:        ts,
		ComponentPayload: nil,
		Message:          testutil.SpanMessage,
	}
	spans.Spans = append(spans.Spans, span)

	return spans, nil
}

func TestAddSpans(t *testing.T) {
	cache := spanCache{}
	spans, err := getTestSpans()
	if err != nil {
		t.Error("Unable to make test spans: %s", err)
	}
	cache.addSpans(spans.Spans)
	if len(cache.spans) != len(spans.Spans) {
		t.Error("Failed to add span to cache")
	}
	cache.Lock()
	cache.Unlock()
}

func TestCommandNotifyNotTrace(t *testing.T) {
	c := Collector{}
	c.CommandNotify("uuid-configure", ssntp.CONFIGURE, &ssntp.Frame{})
	testutil.PollResult(func() bool {
		if len(c.cache.spans) != 0 {
			t.Error("Non ssntp.Trace command handled")
		}
		return false
	}, 10 * time.Millisecond)
	if len(c.cache.spans) != 0 {
		t.Error("Non ssntp.TRACE command handled")
	}
}

func TestCommandNotify(t *testing.T) {
	c := Collector{}
	spans, err := getTestSpans()
	if err != nil {
		t.Error("Unable to make test spans: %s", err)
	}
	frame := ssntp.Frame{}
	frame.Payload, err = yaml.Marshal(&spans)
	if err != nil {
		t.Error("Unable to marshal span to payload yaml")
	}
	c.CommandNotify("uuid-trace", ssntp.TRACE, &frame)
	testutil.PollResult(func() bool {
		defer c.cache.Unlock()
		c.cache.Lock()
		if len(c.cache.spans) != 1 {
			return false
		}
		if c.cache.spans[0].UUID != spans.Spans[0].UUID {
			t.Error("ssntp.TRACE command not handled")
		}
		return true
	}, 10 * time.Millisecond)
}

func TestNewCollectorDefaultStore(t *testing.T) {
	config := CollectorConfig{}
	config.Port = 1
	config.CAcert = "fakecacert"
	config.Cert = "fakecert"
	collector, err := NewCollector(&config)
	if err != nil {
		t.Error("Unable to create collector with default store")
	}
	if reflect.TypeOf(collector.store) != reflect.TypeOf(&Noop{}) {
		t.Error("Collector store not set to no op default")
	}
	if collector.port != 1 {
		t.Error("Collector unexpectedly reset port")
	}
	if collector.caCert != "fakecacert" {
		t.Error("Collector unexpectedly reset cacert")
	}
	if collector.cert != "fakecert" {
		t.Error("Collector unexpectedly reset cert")
	}
}

type testStore struct {
}

func (n *testStore) Store(span payloads.Span) error {
	return nil
}


func TestNewCollectorDefaultPort(t *testing.T) {
	config := CollectorConfig{}
	config.Store = &testStore{}
	config.CAcert = "fakecacert"
	config.Cert = "fakecert"
	collector, err := NewCollector(&config)
	if err != nil {
		t.Error("Unable to create collector with default port")
	}
	if reflect.TypeOf(collector.store) != reflect.TypeOf(&testStore{}) {
		t.Error("Collector unexpectedly reset store")
	}
	if collector.port != TracePort {
		t.Error("Collector port not set to default")
	}
	if collector.caCert != "fakecacert" {
		t.Error("Collector unexpectedly reset cacert")
	}
	if collector.cert != "fakecert" {
		t.Error("Collector unexpectedly reset cert")
	}
}

func TestNewCollectorDefaultCAcert(t *testing.T) {
	config := CollectorConfig{}
	config.Store = &testStore{}
	config.Port = 1
	config.Cert = "fakecert"
	_, err := NewCollector(&config)
	if err == nil || err.Error() != "missing CA" {
		t.Error("Expected missing CA error")
	}
}

func TestNewCollectorDefaultCert(t *testing.T) {
	config := CollectorConfig{}
	config.Store = &testStore{}
	config.Port = 1
	config.CAcert = "fakecacert"
	_, err := NewCollector(&config)
	if err == nil || err.Error() != "missing private key" {
		t.Error("Expected missing private key error")
	}
}
