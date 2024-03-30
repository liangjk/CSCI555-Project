package session

type Err int

const (
	OK Err = iota + 1
	WrongLeader
	RepeatedRequest
	LockNotExist
	LockBusy
	LockNotAcquired
	LockReacquire
	Terminated
)

type PathArgs struct {
	Path   string
	Sessid int32
	Seq    int64
}

type ErrReply struct {
	Result Err
	Lease  int64
}
