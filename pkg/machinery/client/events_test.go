// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"

	machineapi "github.com/chubo-dev/chubo/pkg/machinery/api/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/proto"
)

func TestUnmarshalEventRuntimeTypeURLPrefixes(t *testing.T) {
	t.Parallel()

	payload := &machineapi.PhaseEvent{}
	fullName := string(payload.ProtoReflect().Descriptor().FullName())

	rawPayload, err := proto.Marshal(payload)
	require.NoError(t, err)

	for _, prefix := range []string{legacyRuntimeEventTypeURLPrefix, chuboRuntimeEventTypeURLPrefix} {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()

			typeURL := prefix + fullName

			event, err := UnmarshalEvent(&machineapi.Event{
				Data: &anypb.Any{
					TypeUrl: typeURL,
					Value:   rawPayload,
				},
				Id:      "event-id",
				ActorId: "actor-id",
			})
			require.NoError(t, err)
			require.NotNil(t, event)
			require.Equal(t, typeURL, event.TypeURL)
			require.Equal(t, "event-id", event.ID)
			require.Equal(t, "actor-id", event.ActorID)
			require.IsType(t, &machineapi.PhaseEvent{}, event.Payload)
		})
	}
}

func TestUnmarshalEventUnsupportedPrefix(t *testing.T) {
	t.Parallel()

	payload := &machineapi.PhaseEvent{}
	fullName := string(payload.ProtoReflect().Descriptor().FullName())

	rawPayload, err := proto.Marshal(payload)
	require.NoError(t, err)

	_, err = UnmarshalEvent(&machineapi.Event{
		Data: &anypb.Any{
			TypeUrl: "custom/runtime/" + fullName,
			Value:   rawPayload,
		},
	})
	require.Error(t, err)

	var unsupportedErr EventNotSupportedError
	require.ErrorAs(t, err, &unsupportedErr)
	require.Equal(t, "custom/runtime/"+fullName, unsupportedErr.TypeURL)
}
