// Copyright 2009 The Go9p Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package srv

import (
	"github.com/jsouthworth/ixp"
)

func (srv *Srv) version(req *Req) {
	tc := req.Tc
	conn := req.Conn

	if tc.Msize < ixp.IOHDRSZ {
		req.RespondError(&ixp.Error{"msize too small", ixp.EINVAL})
		return
	}

	if tc.Msize < conn.Msize {
		conn.Msize = tc.Msize
	}

	conn.Dotu = tc.Version == "9P2000.u" && srv.Dotu
	ver := "9P2000"
	if conn.Dotu {
		ver = "9P2000.u"
	}

	/* make sure that the responses of all current requests will be ignored */
	conn.Lock()
	for tag, r := range conn.reqs {
		if tag == ixp.NOTAG {
			continue
		}

		for rr := r; rr != nil; rr = rr.next {
			rr.Lock()
			rr.status |= reqFlush
			rr.Unlock()
		}
	}
	conn.Unlock()

	req.RespondRversion(conn.Msize, ver)
}

func (srv *Srv) auth(req *Req) {
	tc := req.Tc
	conn := req.Conn
	if tc.Afid == ixp.NOFID {
		req.RespondError(Eunknownfid)
		return
	}

	req.Afid = conn.FidNew(tc.Afid)
	if req.Afid == nil {
		req.RespondError(Einuse)
		return
	}

	var user ixp.User = nil
	if tc.Unamenum != ixp.NOUID || conn.Dotu {
		user = srv.Upool.Uid2User(int(tc.Unamenum))
	} else if tc.Uname != "" {
		user = srv.Upool.Uname2User(tc.Uname)
	}

	if user == nil {
		req.RespondError(Enouser)
		return
	}

	req.Afid.User = user
	req.Afid.Type = ixp.QTAUTH
	if aop, ok := (srv.ops).(AuthOps); ok {
		aqid, err := aop.AuthInit(req.Afid, tc.Aname)
		if err != nil {
			req.RespondError(err)
		} else {
			aqid.Type |= ixp.QTAUTH // just in case
			req.RespondRauth(aqid)
		}
	} else {
		req.RespondError(Enoauth)
	}

}

func (srv *Srv) authPost(req *Req) {
	if req.Rc != nil && req.Rc.Type == ixp.Rauth {
		req.Afid.IncRef()
	}
}

func (srv *Srv) attach(req *Req) {
	tc := req.Tc
	conn := req.Conn
	if tc.Fid == ixp.NOFID {
		req.RespondError(Eunknownfid)
		return
	}

	req.Fid = conn.FidNew(tc.Fid)
	if req.Fid == nil {
		req.RespondError(Einuse)
		return
	}

	if tc.Afid != ixp.NOFID {
		req.Afid = conn.FidGet(tc.Afid)
		if req.Afid == nil {
			req.RespondError(Eunknownfid)
		}
	}

	var user ixp.User = nil
	if tc.Unamenum != ixp.NOUID || conn.Dotu {
		user = srv.Upool.Uid2User(int(tc.Unamenum))
	} else if tc.Uname != "" {
		user = srv.Upool.Uname2User(tc.Uname)
	}

	if user == nil {
		req.RespondError(Enouser)
		return
	}

	req.Fid.User = user
	if aop, ok := (srv.ops).(AuthOps); ok {
		err := aop.AuthCheck(req.Fid, req.Afid, tc.Aname)
		if err != nil {
			req.RespondError(err)
			return
		}
	}

	(srv.ops).(ReqOps).Attach(req)
}

func (srv *Srv) attachPost(req *Req) {
	if req.Rc != nil && req.Rc.Type == ixp.Rattach {
		req.Fid.Type = req.Rc.Qid.Type
		req.Fid.IncRef()
	}
}

func (srv *Srv) flush(req *Req) {
	conn := req.Conn
	tag := req.Tc.Oldtag
	ixp.PackRflush(req.Rc)
	conn.Lock()
	r := conn.reqs[tag]
	if r != nil {
		req.flushreq = r.flushreq
		r.flushreq = req
	}
	conn.Unlock()

	if r == nil {
		// there are no requests with that tag
		req.Respond()
		return
	}

	r.Lock()
	status := r.status
	if (status & (reqWork | reqSaved)) == 0 {
		/* the request is not worked on yet */
		r.status |= reqFlush
	}
	r.Unlock()

	if (status & (reqWork | reqSaved)) == 0 {
		r.Respond()
	} else {
		if op, ok := (srv.ops).(FlushOp); ok {
			op.Flush(r)
		}
	}
}

func (srv *Srv) walk(req *Req) {
	conn := req.Conn
	tc := req.Tc
	fid := req.Fid

	/* we can't walk regular files, only clone them */
	if len(tc.Wname) > 0 && (fid.Type&ixp.QTDIR) == 0 {
		req.RespondError(Enotdir)
		return
	}

	/* we can't walk open files */
	if fid.opened {
		req.RespondError(Ebaduse)
		return
	}

	if tc.Fid != tc.Newfid {
		req.Newfid = conn.FidNew(tc.Newfid)
		if req.Newfid == nil {
			req.RespondError(Einuse)
			return
		}

		req.Newfid.User = fid.User
		req.Newfid.Type = fid.Type
	} else {
		req.Newfid = req.Fid
		req.Newfid.IncRef()
	}

	(req.Conn.Srv.ops).(ReqOps).Walk(req)
}

func (srv *Srv) walkPost(req *Req) {
	rc := req.Rc
	if rc == nil || rc.Type != ixp.Rwalk || req.Newfid == nil {
		return
	}

	n := len(rc.Wqid)
	if n > 0 {
		req.Newfid.Type = rc.Wqid[n-1].Type
	} else {
		req.Newfid.Type = req.Fid.Type
	}

	// Don't retain the fid if only a partial walk succeeded
	if n != len(req.Tc.Wname) {
		return
	}

	if req.Newfid.fid != req.Fid.fid {
		req.Newfid.IncRef()
	}
}

func (srv *Srv) open(req *Req) {
	fid := req.Fid
	tc := req.Tc
	if fid.opened {
		req.RespondError(Eopen)
		return
	}

	if (fid.Type&ixp.QTDIR) != 0 && tc.Mode != ixp.OREAD {
		req.RespondError(Eperm)
		return
	}

	fid.Omode = tc.Mode
	(req.Conn.Srv.ops).(ReqOps).Open(req)
}

func (srv *Srv) openPost(req *Req) {
	if req.Fid != nil {
		req.Fid.opened = req.Rc != nil && req.Rc.Type == ixp.Ropen
	}
}

func (srv *Srv) create(req *Req) {
	fid := req.Fid
	tc := req.Tc
	if fid.opened {
		req.RespondError(Eopen)
		return
	}

	if (fid.Type & ixp.QTDIR) == 0 {
		req.RespondError(Enotdir)
		return
	}

	/* can't open directories for other than reading */
	if (tc.Perm&ixp.DMDIR) != 0 && tc.Mode != ixp.OREAD {
		req.RespondError(Eperm)
		return
	}

	/* can't create special files if not 9P2000.u */
	if (tc.Perm&(ixp.DMNAMEDPIPE|ixp.DMSYMLINK|ixp.DMLINK|ixp.DMDEVICE|ixp.DMSOCKET)) != 0 && !req.Conn.Dotu {
		req.RespondError(Eperm)
		return
	}

	fid.Omode = tc.Mode
	(req.Conn.Srv.ops).(ReqOps).Create(req)
}

func (srv *Srv) createPost(req *Req) {
	if req.Rc != nil && req.Rc.Type == ixp.Rcreate && req.Fid != nil {
		req.Fid.Type = req.Rc.Qid.Type
		req.Fid.opened = true
	}
}

func (srv *Srv) read(req *Req) {
	tc := req.Tc
	fid := req.Fid
	if tc.Count+ixp.IOHDRSZ > req.Conn.Msize {
		req.RespondError(Etoolarge)
		return
	}

	if (fid.Type & ixp.QTAUTH) != 0 {
		var n int

		rc := req.Rc
		err := ixp.InitRread(rc, tc.Count)
		if err != nil {
			req.RespondError(err)
			return
		}

		if op, ok := (req.Conn.Srv.ops).(AuthOps); ok {
			n, err = op.AuthRead(fid, tc.Offset, rc.Data)
			if err != nil {
				req.RespondError(err)
				return
			}

			ixp.SetRreadCount(rc, uint32(n))
			req.Respond()
		} else {
			req.RespondError(Enotimpl)
		}

		return
	}

	if (fid.Type & ixp.QTDIR) != 0 {
		if tc.Offset == 0 {
			fid.Diroffset = 0
		} else if tc.Offset != fid.Diroffset {
			req.RespondError(Ebadoffset)
			return
		}
	}

	(req.Conn.Srv.ops).(ReqOps).Read(req)
}

func (srv *Srv) readPost(req *Req) {
	if req.Rc != nil && req.Rc.Type == ixp.Rread && (req.Fid.Type&ixp.QTDIR) != 0 {
		req.Fid.Diroffset += uint64(req.Rc.Count)
	}
}

func (srv *Srv) write(req *Req) {
	fid := req.Fid
	tc := req.Tc
	if (fid.Type & ixp.QTAUTH) != 0 {
		tc := req.Tc
		if op, ok := (req.Conn.Srv.ops).(AuthOps); ok {
			n, err := op.AuthWrite(req.Fid, tc.Offset, tc.Data)
			if err != nil {
				req.RespondError(err)
			} else {
				req.RespondRwrite(uint32(n))
			}
		} else {
			req.RespondError(Enotimpl)
		}

		return
	}

	if !fid.opened || (fid.Type&ixp.QTDIR) != 0 || (fid.Omode&3) == ixp.OREAD {
		req.RespondError(Ebaduse)
		return
	}

	if tc.Count+ixp.IOHDRSZ > req.Conn.Msize {
		req.RespondError(Etoolarge)
		return
	}

	(req.Conn.Srv.ops).(ReqOps).Write(req)
}

func (srv *Srv) clunk(req *Req) {
	fid := req.Fid
	if (fid.Type & ixp.QTAUTH) != 0 {
		if op, ok := (req.Conn.Srv.ops).(AuthOps); ok {
			op.AuthDestroy(fid)
			req.RespondRclunk()
		} else {
			req.RespondError(Enotimpl)
		}

		return
	}

	(req.Conn.Srv.ops).(ReqOps).Clunk(req)
}

func (srv *Srv) clunkPost(req *Req) {
	if req.Rc != nil && req.Rc.Type == ixp.Rclunk && req.Fid != nil {
		req.Fid.DecRef()
	}
}

func (srv *Srv) remove(req *Req) { (req.Conn.Srv.ops).(ReqOps).Remove(req) }

func (srv *Srv) removePost(req *Req) {
	if req.Rc != nil && req.Fid != nil {
		req.Fid.DecRef()
	}
}

func (srv *Srv) stat(req *Req) { (req.Conn.Srv.ops).(ReqOps).Stat(req) }

func (srv *Srv) wstat(req *Req) {
	/*
		fid := req.Fid
		d := &req.Tc.Dir
		if d.Type != uint16(0xFFFF) || d.Dev != uint32(0xFFFFFFFF) || d.Version != uint32(0xFFFFFFFF) ||
			d.Path != uint64(0xFFFFFFFFFFFFFFFF) {
			req.RespondError(Eperm)
			return
		}

		if (d.Mode != 0xFFFFFFFF) && (((fid.Type&ixp.QTDIR) != 0 && (d.Mode&ixp.DMDIR) == 0) ||
			((d.Type&ixp.QTDIR) == 0 && (d.Mode&ixp.DMDIR) != 0)) {
			req.RespondError(Edirchange)
			return
		}
	*/

	(req.Conn.Srv.ops).(ReqOps).Wstat(req)
}
