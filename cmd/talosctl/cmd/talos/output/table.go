//go:build !chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/state"
)

// Table outputs resources in Table view.
type Table struct {
	w           tabwriter.Writer
	withEvents  bool
	displayType string
}

// NewTable initializes table resource output.
func NewTable(writer io.Writer) *Table {
	output := &Table{}
	output.w.Init(writer, 0, 0, 3, ' ', 0)

	return output
}

// WriteHeader implements output.Writer interface.
func (table *Table) WriteHeader(definition *meta.ResourceDefinition, withEvents bool) error {
	table.withEvents = withEvents
	fields := []string{"NAMESPACE", "TYPE", "ID", "VERSION"}

	if withEvents {
		fields = slices.Insert(fields, 0, "*")
	}

	table.displayType = definition.TypedSpec().DisplayType

	fields = slices.Insert(fields, 0, "NODE")

	_, err := fmt.Fprintln(&table.w, strings.Join(fields, "\t"))

	return err
}

// WriteResource implements output.Writer interface.
func (table *Table) WriteResource(node string, r resource.Resource, event state.EventType) error {
	values := []string{r.Metadata().Namespace(), table.displayType, r.Metadata().ID(), r.Metadata().Version().String()}

	if table.withEvents {
		var label string

		switch event {
		case state.Created:
			label = "+"
		case state.Destroyed:
			label = "-"
		case state.Updated:
			label = " "
		case state.Bootstrapped, state.Errored, state.Noop:
			return nil
		}

		values = slices.Insert(values, 0, label)
	}

	values = slices.Insert(values, 0, node)

	_, err := fmt.Fprintln(&table.w, strings.Join(values, "\t"))

	return err
}

// Flush implements output.Writer interface.
func (table *Table) Flush() error {
	return table.w.Flush()
}
