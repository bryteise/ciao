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
	"flag"
	// "fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
	"github.com/01org/ciao/testutil"
)

var integration bool
func init() {
	flag.BoolVar(&integration, "integration", false, "Set to true when running integration tests")
}

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
		return
	}

	if _, ok := <-tracer.stopChannel; ok {
		t.Error("Channel is not closed")
		return
	}
}

func TestStopNotRunning(t *testing.T) {
	tracer := Tracer{}
	// Should not panic if status is stopped
	tracer.status.status = stopped

	tracer.Stop()
}

func validateNewTracer(tracer *Tracer, config TracerConfig, t *testing.T) {
	if tracer.ssntpUUID != config.UUID {
		t.Error("New tracer didn't get correct ssntpUUID")
	}
	if tracer.component != config.Component {
		if config.Component != Anonymous {
			t.Error("New tracer didn't get correct component")
		}
	}
	if tracer.spanner != config.Spanner {
		if config.Spanner != nil {
			t.Error("New tracer didn't get correct spanner")
		}
	}
	if tracer.collectorURI != config.CollectorURI {
		if config.CollectorURI != "" {
			t.Error("New tracer didn't get correct collectorURI")
		}
	}
	if tracer.caCert != config.CAcert {
		t.Error("New tracer didn't get correct caCert")
	}
	if tracer.cert != config.Cert {
		t.Error("New tracer didn't get correct cert")
	}
	if config.Port != 1 && config.Port != TracePort {
		t.Error("New tracer didn't get correct port")
	}
}

// NewTracer needs setup for ssntp.Dial (called by dailAndListen)
type traceMockNotifier struct {}
func (server *traceMockNotifier) ConnectNotify(uuid string, role ssntp.Role) {
}
func (server *traceMockNotifier) DisconnectNotify(uuid string, role ssntp.Role) {
}
func (server *traceMockNotifier) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
}
func (server *traceMockNotifier) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
}
func (server *traceMockNotifier) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
}
func (server *traceMockNotifier) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
}

func newTracerDefaultRunner(part string, t *testing.T) {
	config := TracerConfig{}
	configFile := testutil.GetConfigFile("tracer-test")
	caPath := "tracer-test-test-ca"
	serverCertPath := "tracer-test-server-test-cert"
	clientCertPath := "tracer-test-client-test-cert"

	template, err := certs.CreateCertTemplate(ssntp.SCHEDULER, "test", "test@test.test", []string{"localhost"}, []string{"127.0.0.1"})
	if err != nil {
		t.Error("Unable to create cert template")
		return
	}
	caWriter, err := os.Create(caPath)
	if err != nil {
		t.Error("Unable to create ca file")
		return
	}
	serverCertWriter, err := os.Create(serverCertPath)
	if err != nil {
		caWriter.Close()
		os.Remove(caPath)
		t.Error("Unable to create server cert file")
		return
	}
	clientCertWriter, err := os.Create(clientCertPath)
	if err != nil {
		caWriter.Close()
		os.Remove(caPath)
		serverCertWriter.Close()
		os.Remove(serverCertPath)
		t.Error("Unable to create client cert file")
		return
	}
	defer caWriter.Close()
	defer os.Remove(caPath)
	defer serverCertWriter.Close()
	defer os.Remove(serverCertPath)
	defer clientCertWriter.Close()
	defer os.Remove(clientCertPath)

	err = certs.CreateServerCert(template, false, serverCertWriter, caWriter)
	if err != nil {
		t.Error("Unable to create server cert")
		return
	}
	serverCert, err := ioutil.ReadFile(serverCertPath)
	if err != nil {
		t.Error("Unable to read server cert")
		return
	}
	err = certs.CreateClientCert(template, false, serverCert, clientCertWriter)
	if err != nil {
		t.Error("Unable to create client cert")
	}

	config.UUID = "uuid"
	config.CAcert = caPath
	config.Cert = clientCertPath
	config.Component = SSNTP
	config.Spanner = testSpan{}
	config.Port = 1
	config.CollectorURI = "127.0.0.1"
	if part == "component" {
		config.Component = ""
	} else if part == "spanner" {
		config.Spanner = nil
	} else if part == "port" {
		config.Port = 0
	} else if part == "collectoruri" {
		config.CollectorURI = ""
	}
	ssntpConfig := ssntp.Config{}
	ssntpConfig.CAcert = caPath
	ssntpConfig.Cert = serverCertPath
	ssntpConfig.ConfigURI = configFile
	ssntpConfig.UUID = "suuid"
	ssntpConfig.Log = ssntp.Log
	serv := ssntp.Server{}
	serv.ServeThreadSync(&ssntpConfig, &traceMockNotifier{})

	tracer, tracerContext, err := NewTracer(&config)
	if err != nil {
		t.Error("Error creating New Tracer")
		return
	}
	validateNewTracer(tracer, config, t)
	if tracerContext.parentUUID != nullUUID {
		t.Error("New tracer didn't set correct TracerContext")
	}
	tracer.Stop()
	serv.Stop()
}

func TestNewTracerDefaultComponent(t *testing.T) {
	newTracerDefaultRunner("component", t)
}

func TestNewTracerDefaultSpanner(t *testing.T) {
	newTracerDefaultRunner("spanner", t)
}

func TestNewTracerDefaultPort(t *testing.T) {
	newTracerDefaultRunner("port", t)
}

func TestNewTracerDefaultCollectorURI(t *testing.T) {
	newTracerDefaultRunner("collectoruri", t)
}

// func TestPushSpan(t *testing.T) {
// 	config := TracerConfig{}
// 	configFile := testutil.GetConfigFile("tracer-test")

// 	config.UUID = "uuid"
// 	cacert, agentCert, err := testutil.RoleToTestCertPath(ssntp.AGENT)
// 	if err != nil {
// 		t.Error("Missing agent test cert")
// 		return
// 	}
// 	config.CAcert = cacert
// 	config.Cert = agentCert

// 	cacert, schedulerCert, err := testutil.RoleToTestCertPath(ssntp.SCHEDULER)
// 	if err != nil {
// 		t.Error("Missing scheduler test cert")
// 	}
// 	ssntpConfig := ssntp.Config{}
// 	ssntpConfig.CAcert = cacert
// 	ssntpConfig.Cert = schedulerCert
// 	ssntpConfig.ConfigURI = configFile
// 	ssntpConfig.UUID = "suuid"
// 	serv := ssntp.Server{}
// 	serv.ServeThreadSync(&ssntpConfig, &traceMockNotifier{})

// 	tracer, _, err := NewTracer(&config)
// 	if err != nil {
// 		t.Error("Error creating New Tracer")
// 		return
// 	}

// 	span := payloads.Span{"uuid", "puuid", "cuuid", time.Time{}, "comp", []byte{0}, "msg"}
// 	err = tracer.pushSpan(span)
// 	if err != nil {
// 		t.Error("Error running pushSpan")
// 	}

// 	tracer.Stop()
// 	serv.Stop()
// }
