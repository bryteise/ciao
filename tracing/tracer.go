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

	"github.com/docker/distribution/uuid"
)

// Component is a tracing identifier for each ciao component.
type Component string

const (
	// Anonymous is a special component for anonymous tracing.
	// Anonymous tracing can only carry span messages but not
	// component specific payloads.
	Anonymous Component = "anonymous"

	// SSNTP is for tracing SSNTP traffic.
	SSNTP Component = "ssntp"

	// Libsnnet is for tracing ciao's networking.
	Libsnnet Component = "libsnnet"
)

const nullUUID = "00000000-0000-0000-0000-000000000000"
const spanChannelDepth = 256

// Tracer is a handle to a ciao tracing agent that will collect
// local spans and send them back to ciao trace collectors.
type Tracer struct {
	ssntpUUID uuid.UUID
	component Component
	spanner   Spanner

	spanChannel chan Span
}

// Context is an opaque structure that gets passed to Trace()
// calls in order to link spans together.
// If you want to link a span A to span B, you should pass the
// trace context returned when calling Trace() to create span A to
// the Trace() call for creating span B.
type Context struct {
	parentUUID uuid.UUID
}

// NewComponentTracer creates a tracer for a specific component.
func NewComponentTracer(component Component, spanner Spanner, ssntpuuid string) (*Tracer, *Context, error) {
	rootUUID, err := uuid.Parse(nullUUID)
	if err != nil {
		return nil, nil, err
	}

	ssntpUUID, err := uuid.Parse(ssntpuuid)
	if err != nil {
		return nil, nil, err
	}

	tracer := Tracer{
		ssntpUUID:   ssntpUUID,
		component:   component,
		spanner:     spanner,
		spanChannel: make(chan Span, spanChannelDepth),
	}

	traceContext := Context{
		parentUUID: rootUUID,
	}

	go tracer.spanListener()

	return &tracer, &traceContext, nil
}

// NewTracer creates an anonymous tracer.
// uuid is the SSNTP UUID of the caller.
func NewTracer(uuid string) (*Tracer, *Context, error) {
	return NewComponentTracer(Anonymous, AnonymousSpanner{}, uuid)
}

func (t *Tracer) spanListener() {
	for {
		select {
		case span := <-t.spanChannel:
			// TODO Send spans to collectors
			fmt.Printf("SPAN: %s\n", span)
		}
	}
}

// Stop will stop a tracer.
// Spans will no longer be listened for and thus won't make
// it up to a trace collector.
func (t *Tracer) Stop() {
}
