// Copyright 2018 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uploader

import (
	"context"
	"io"
)

// Uploader is an interface used to upload objects to remote object store
// servers.
type Uploader interface {
	Upload(Options) error
	Delete(Options) error
}

// Options presents the set of options to the Upload method.
type Options struct {
	Context      context.Context
	Bucket       string
	Path         string
	Body         io.Reader
	ContentType  *string
	CacheControl *string
}
