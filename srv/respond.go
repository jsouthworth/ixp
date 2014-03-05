// Copyright 2009 The Go9p Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package srv

import "fmt"
import "github.com/jsouthworth/ixp"

// Respond to the request with Rerror message
func (req *Req) RespondError(err interface{}) {
	switch e := err.(type) {
	case *ixp.Error:
		ixp.PackRerror(req.Rc, e.Error(), uint32(e.Errornum), req.Conn.Dotu)
	case error:
		ixp.PackRerror(req.Rc, e.Error(), uint32(ixp.EIO), req.Conn.Dotu)
	default:
		ixp.PackRerror(req.Rc, fmt.Sprintf("%v", e), uint32(ixp.EIO), req.Conn.Dotu)
	}

	req.Respond()
}

// Respond to the request with Rversion message
func (req *Req) RespondRversion(msize uint32, version string) {
	err := ixp.PackRversion(req.Rc, msize, version)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rauth message
func (req *Req) RespondRauth(aqid *ixp.Qid) {
	err := ixp.PackRauth(req.Rc, aqid)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rflush message
func (req *Req) RespondRflush() {
	err := ixp.PackRflush(req.Rc)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rattach message
func (req *Req) RespondRattach(aqid *ixp.Qid) {
	err := ixp.PackRattach(req.Rc, aqid)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rwalk message
func (req *Req) RespondRwalk(wqids []ixp.Qid) {
	err := ixp.PackRwalk(req.Rc, wqids)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Ropen message
func (req *Req) RespondRopen(qid *ixp.Qid, iounit uint32) {
	err := ixp.PackRopen(req.Rc, qid, iounit)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rcreate message
func (req *Req) RespondRcreate(qid *ixp.Qid, iounit uint32) {
	err := ixp.PackRcreate(req.Rc, qid, iounit)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rread message
func (req *Req) RespondRread(data []byte) {
	err := ixp.PackRread(req.Rc, data)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rwrite message
func (req *Req) RespondRwrite(count uint32) {
	err := ixp.PackRwrite(req.Rc, count)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rclunk message
func (req *Req) RespondRclunk() {
	err := ixp.PackRclunk(req.Rc)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rremove message
func (req *Req) RespondRremove() {
	err := ixp.PackRremove(req.Rc)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rstat message
func (req *Req) RespondRstat(st *ixp.Dir) {
	err := ixp.PackRstat(req.Rc, st, req.Conn.Dotu)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}

// Respond to the request with Rwstat message
func (req *Req) RespondRwstat() {
	err := ixp.PackRwstat(req.Rc)
	if err != nil {
		req.RespondError(err)
	} else {
		req.Respond()
	}
}
