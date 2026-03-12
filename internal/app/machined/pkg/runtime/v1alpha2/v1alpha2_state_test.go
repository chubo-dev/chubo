package v1alpha2

import (
	"context"
	"strings"
	"testing"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/safe"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

func TestNewStateRegistersOpenBaoInitResource(t *testing.T) {
	t.Parallel()

	st, err := NewState()
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	rd, err := safe.ReaderGetByID[*meta.ResourceDefinition](
		context.Background(),
		st.Resources(),
		resource.ID(strings.ToLower(string(secrets.OpenBaoInitType))),
	)
	if err != nil {
		t.Fatalf("openbao init resource definition missing: %v", err)
	}

	if got := rd.TypedSpec().Type; got != secrets.OpenBaoInitType {
		t.Fatalf("resource definition type = %q, want %q", got, secrets.OpenBaoInitType)
	}
}
