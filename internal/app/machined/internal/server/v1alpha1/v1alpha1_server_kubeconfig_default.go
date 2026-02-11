// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package runtime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/siderolabs/talos/pkg/kubeconfig"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
)

// Kubeconfig implements the machine.MachineServer interface.
func (s *Server) Kubeconfig(_ *emptypb.Empty, obj machine.MachineService_KubeconfigServer) error {
	if err := s.checkControlplane("kubeconfig"); err != nil {
		return err
	}

	var b bytes.Buffer

	if err := kubeconfig.GenerateAdmin(s.Controller.Runtime().Config().Cluster(), &b); err != nil {
		return err
	}

	// Wrap in .tar.gz to match Copy protocol.
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	tarW := tar.NewWriter(zw)

	err := tarW.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "kubeconfig",
		Size:     int64(b.Len()),
		ModTime:  time.Now(),
		Mode:     0o600,
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(tarW, &b)
	if err != nil {
		return err
	}

	if err = zw.Close(); err != nil {
		return err
	}

	return obj.Send(&common.Data{Bytes: buf.Bytes()})
}
