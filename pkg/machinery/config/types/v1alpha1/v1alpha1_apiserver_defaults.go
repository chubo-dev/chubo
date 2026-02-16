// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

// APIServerDefaultAuditPolicy is used by doc/schema generation as an example
// value for the legacy API server audit policy field.
//
// Chubo OS does not manage this surface, but we keep the symbol until the
// v1alpha1 schema is fully purged.
var APIServerDefaultAuditPolicy = Unstructured{
	Object: map[string]any{},
}
