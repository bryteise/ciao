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

type Component string

const (
	Anonymous Component = "anonymous"
	SSNTP     Component = "ssntp"
	Libsnnet  Component = "libsnnet"
)

const nullUUID = "00000000-0000-0000-0000-000000000000"
const spanChannelDepth = 256

type Tracer struct {
	ssntpUUID uuid.UUID
	component Component
	spanner   Spanner

	spanChannel chan Span
}

type TraceContext struct {
	parentUUID uuid.UUID
}

func NewComponentTracer(component Component, spanner Spanner, ssntpUUID uuid.UUID) (*Tracer, *TraceContext, error) {
	rootUUID, err := uuid.Parse(nullUUID)
	if err != nil {
		return nil, nil, err
	}

	tracer := Tracer{
		ssntpUUID:   ssntpUUID,
		component:   component,
		spanner:     spanner,
		spanChannel: make(chan Span, spanChannelDepth),
	}

	traceContext := TraceContext{
		parentUUID: rootUUID,
	}

	go tracer.spanListener()

	return &tracer, &traceContext, nil
}

func NewTracer(ssntpUUID uuid.UUID) (*Tracer, *TraceContext, error) {
	return NewComponentTracer(Anonymous, AnonymousSpanner{}, ssntpUUID)
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

func (t *Tracer) Stop() {
}
