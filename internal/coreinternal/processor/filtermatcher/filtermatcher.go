// Copyright The OpenTelemetry Authors
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

package filtermatcher // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filtermatcher"

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterset"
)

type instrumentationLibraryMatcher struct {
	Name    filterset.FilterSet
	Version filterset.FilterSet
}

// PropertiesMatcher allows matching a span against various span properties.
type PropertiesMatcher struct {
	// Instrumentation libraries to compare against
	libraries []instrumentationLibraryMatcher

	// The attribute values are stored in the internal format.
	attributes AttributesMatcher

	// The attribute values are stored in the internal format.
	resources AttributesMatcher
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (PropertiesMatcher, error) {
	var lm []instrumentationLibraryMatcher
	for _, library := range mp.Libraries {
		name, err := filterset.CreateFilterSet([]string{library.Name}, &mp.Config)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating library name filters: %w", err)
		}

		var version filterset.FilterSet
		if library.Version != nil {
			filter, err := filterset.CreateFilterSet([]string{*library.Version}, &mp.Config)
			if err != nil {
				return PropertiesMatcher{}, fmt.Errorf("error creating library version filters: %w", err)
			}
			version = filter
		}

		lm = append(lm, instrumentationLibraryMatcher{Name: name, Version: version})
	}

	var err error
	var am AttributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = NewAttributesMatcher(mp.Config, mp.Attributes)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating attribute filters: %w", err)
		}
	}

	var rm AttributesMatcher
	if len(mp.Resources) > 0 {
		rm, err = NewAttributesMatcher(mp.Config, mp.Resources)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating resource filters: %w", err)
		}
	}

	return PropertiesMatcher{
		libraries:  lm,
		attributes: am,
		resources:  rm,
	}, nil
}

// Match matches a span or log to a set of properties.
func (mp *PropertiesMatcher) Match(attributes pcommon.Map, resource pcommon.Resource, library pcommon.InstrumentationScope) bool {
	for _, matcher := range mp.libraries {
		if !matcher.Name.Matches(library.Name()) {
			return false
		}
		if matcher.Version != nil && !matcher.Version.Matches(library.Version()) {
			return false
		}
	}

	if mp.resources != nil && !mp.resources.Match(resource.Attributes()) {
		return false
	}

	return mp.attributes.Match(attributes)
}
