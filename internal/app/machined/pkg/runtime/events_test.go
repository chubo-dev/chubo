// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
)

func TestNewEventTypeURL(t *testing.T) {
	t.Parallel()

	event := NewEvent(&machine.SequenceEvent{}, "actor-1")

	require.Equal(t, "chubo/runtime/machine.SequenceEvent", event.TypeURL)
	require.Equal(t, "actor-1", event.ActorID)
}

func TestNewEventTypeURLForNilPayload(t *testing.T) {
	t.Parallel()

	event := NewEvent(nil, "actor-1")

	require.Empty(t, event.TypeURL)
}

func TestEventToMachineEvent(t *testing.T) {
	t.Parallel()

	event := NewEvent(&machine.SequenceEvent{}, "actor-1")

	wireEvent, err := event.ToMachineEvent()
	require.NoError(t, err)
	require.Equal(t, event.TypeURL, wireEvent.GetData().GetTypeUrl())
	require.Equal(t, event.ActorID, wireEvent.GetActorId())
}
