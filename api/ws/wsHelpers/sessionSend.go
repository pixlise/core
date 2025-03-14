package wsHelpers

import (
	"github.com/olahol/melody"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
)

func SendForSession(s *melody.Session, wsmsg *protos.WSMessage) {
	bytes, err := proto.Marshal(wsmsg)
	if err != nil {
		s.CloseWithMsg([]byte("--" + err.Error()))
		return
	}

	s.WriteBinary(bytes)
}
