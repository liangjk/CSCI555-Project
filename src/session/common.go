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

type Opcode int

const (
	Create Opcode = iota + 1
	Remove
	Acquire
	Release
	Extend
	Revoke
)

type Op struct {
	Code   Opcode
	Path   string
	Sessid int32
	Seq    int64
	Lease  int64
}

const (
	DefaultLease = 12
	GracePeriod  = DefaultLease / 2
)

type File struct {
	Sessid int32
	Lease  int64
}
