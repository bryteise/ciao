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
	"testing"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
)

type testSpan struct {
}

func (s testSpan) Span(context interface{}) []byte {
	return nil
}

func TestNewTracerDefaultUUID(t *testing.T) {
	config := TracerConfig{}

	config.CAcert = "fakecacert"
	config.Cert = "fakecert"
	config.Component = SSNTP
	config.Spanner = testSpan{}
	config.Port = 1
	config.CollectorURI = "fakeuri"
	_, _, err := NewTracer(&config)
	if err == nil || err.Error() != "empty SSNTP UUID" {
		t.Error("Expected empty SSNTP UUID error")
	}
}

func TestNewTracerDefaultCAcert(t *testing.T) {
	config := TracerConfig{}

	config.UUID = "uuid"
	config.Cert = "fakecert"
	config.Component = SSNTP
	config.Spanner = testSpan{}
	config.Port = 1
	config.CollectorURI = "fakeuri"
	_, _, err := NewTracer(&config)
	if err == nil || err.Error() != "missing CA" {
		t.Error("Expected missing CA error")
	}
}

func TestNewTracerDefaultCert(t *testing.T) {
	config := TracerConfig{}

	config.UUID = "uuid"
	config.CAcert = "fakecacert"
	config.Component = SSNTP
	config.Spanner = testSpan{}
	config.Port = 1
	config.CollectorURI = "fakeuri"
	_, _, err := NewTracer(&config)
	if err == nil || err.Error() != "missing private key" {
		t.Error("Expected missing private key error")
	}
}

func TestQueueSpanNoPush(t *testing.T) {
	tracer := Tracer{}

	tracer.queueSpan(payloads.Span{})
	if len(tracer.spanQueue) != 1 {
		t.Error("Failed to queue span")
	}
}

func TestSpanListener(t *testing.T) {
	tracer := Tracer{}
	tracer.spanChannel = make(chan payloads.Span, spanChannelDepth)
	tracer.stopChannel = make(chan struct{})
	tracer.statusChannel = make(chan status)
	tracer.status.status = stopped

	go tracer.spanListener()

	select {
	case status := <-tracer.statusChannel:
		if status != running {
			t.Error("Listener couldn't start")
		}

	case <-time.After(10 * time.Millisecond):
		tracer.Stop()
		t.Error("Did not receive a tracer status")
	}

	tracer.spanChannel <- payloads.Span{}
	if !testutil.PollResult(func() bool {
		defer tracer.status.Unlock()
		tracer.status.Lock()
		if tracer.status.status != running {
			return false
		}

		defer tracer.spanQueueLock.Unlock()
		tracer.spanQueueLock.Lock()
		if len(tracer.spanQueue) == 1 {
			return true
		}
		return false
	}, 10 * time.Millisecond) {
		t.Error("Listener failed to queue span")
	}
	close(tracer.stopChannel)
}

func TestStop(t *testing.T) {
	tracer := Tracer{}
	tracer.stopChannel = make(chan struct{})
	tracer.status.status = running

	tracer.Stop()

	if tracer.status.status != stopped {
		t.Error("Stop didn't update channel status correctly")
	}

	if _, ok := <-tracer.stopChannel; ok {
		t.Error("Channel is not closed")
	}
}

func TestStopNotRunning(t *testing.T) {
	tracer := Tracer{}
	// Should not panic if status is stopped
	tracer.status.status = stopped

	tracer.Stop()
}
