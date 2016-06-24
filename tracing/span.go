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
	"time"

	"github.com/01org/ciao/ssntp/uuid"
)

// Span is the ciao tracing singleton.
// A Span will carry a trace message together with an
// optional component specific binary payload.
// Spans can be linked together and eventually form a
// tracing tree.
type Span struct {
	uuid        uuid.UUID
	parentUUID  uuid.UUID
	creatorUUID uuid.UUID

	timestamp        time.Time
	component        Component
	componentPayload []byte
	message          string
}

// Spanner is a span interface for components to add their specific
// binary payloads to any given Span.
type Spanner interface {
	// Span takes a component specific structure and returns a binary payload.
	Span(componentContext interface{}) []byte
}

// String formats a span into a Go string.
func (s Span) String() string {
	return fmt.Sprintf("\n\tSpan UUID [%s]\n\tParent UUID [%s]\n\tTimestamp [%v]\n\tComponent [%s]\n\tMessage [%s]\n",
		s.uuid, s.parentUUID, s.timestamp, s.component, s.message)
}

// AnonymousSpanner is the Anonymous component Spanner implementation
type AnonymousSpanner struct{}

// Span returns a nil payload for Anonymous components.
func (s AnonymousSpanner) Span(context interface{}) []byte {
	return nil
}
