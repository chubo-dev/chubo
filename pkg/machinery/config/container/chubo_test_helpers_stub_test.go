// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package container_test

import "github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"

func applyChuboOSValidationDefaults(*v1alpha1.Config) {}

