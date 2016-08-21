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

package payloads_test

import (
	"testing"
	"time"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"github.com/01org/ciao/trace"
	"gopkg.in/yaml.v2"
)

func TestSpanMarshal(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, testutil.SpanTimeStamp)
	if err != nil {
		t.Error(err)
	}

	span := Span{
		UUID:             testutil.SpanUUID,
		ParentUUID:       testutil.ParentSpanUUID,
		CreatorUUID:      testutil.AgentUUID,
		Component:        string(trace.Anonymous),
		Timestamp:        ts,
		ComponentPayload: nil,
		Message:          testutil.SpanMessage,
	}

	spans := Spans{}
	spans.Spans = append(spans.Spans, span)
	spans.Spans = append(spans.Spans, span)

	y, err := yaml.Marshal(&spans)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.SpansYaml {
		t.Errorf("Ready marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.SpansYaml)
	}
}

func TestSpanUnarshal(t *testing.T) {
	var spans Spans
	err := yaml.Unmarshal([]byte(testutil.SpansYaml), &spans)
	if err != nil {
		t.Error(err)
	}

	ts, err := time.Parse(time.RFC3339, testutil.SpanTimeStamp)
	if err != nil {
		t.Error(err)
	}

	if len(spans.Spans) != 2 {
		t.Errorf("Wrong number of spans: %d spans", len(spans.Spans))
	}

	if !spans.Spans[0].Timestamp.Equal(ts) {
		t.Errorf("Time stamps mismatch %s vs %s", spans.Spans[0].Timestamp, ts)
	}

	if spans.Spans[0].UUID != testutil.SpanUUID {
		t.Errorf("Wrong span UUID %s vs %s", spans.Spans[0].UUID, testutil.SpanUUID)
	}

	if len(spans.Spans[0].ComponentPayload) != 0 {
		t.Errorf("Unknown span payload %v", spans.Spans[0].ComponentPayload)
	}
}
