package ipc

type OpCode int32

const (
	OpHandshake OpCode = 0
	OpFrame     OpCode = 1
	OpActivity  OpCode = 2
	OpReady     OpCode = 3
	OpClose     OpCode = 4

	OpActivityJoin        OpCode = 5
	OpActivitySpectate    OpCode = 6
	OpActivityJoinRequest OpCode = 7
)

type Frame struct {
	Opcode OpCode `json:"op"`
	Data   any    `json:"d"`
}
