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

package payloads

import (
	"fmt"
	"time"
)

// Span is the ciao tracing singleton.
// A Span will carry a trace message together with an
// optional component specific binary payload.
// Spans can be linked together and eventually form a
// tracing tree.
type Span struct {
	UUID        string `yaml:"uuid"`
	ParentUUID  string `yaml:"parent_uuid"`
	CreatorUUID string `yaml:"creator_uuid"`

	Timestamp        time.Time `yaml:"timestamp"`
	Component        string    `yaml:"component"`
	ComponentPayload []byte    `yaml:"payload"`
	Message          string    `yaml:"message"`
}

// String formats a span into a Go string.
func (s Span) String() string {
	return fmt.Sprintf("\n\tSpan UUID [%s]\n\tParent UUID [%s]\n\tTimestamp [%v]\n\tComponent [%s]\n\tMessage [%s]\n",
		s.UUID, s.ParentUUID, s.Timestamp, s.Component, s.Message)
}

// Spans represents an array of Span.
type Spans struct {
	Spans []Span `yaml:"spans"`
}
