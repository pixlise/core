package utils

import (
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func SendProtoBinary(w http.ResponseWriter, m protoreflect.ProtoMessage) {
	//w.Header().Add("Access-Control-Allow-Origin", "*")

	b, err := proto.Marshal(m)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// See: https://stackoverflow.com/questions/30505408/what-is-the-correct-protobuf-content-type
		// Content type could be:
		// "application/octet-stream"
		// "application/protobuf"
		// "application/x-protobuf"
		// "application/vnd.google.protobuf"
		w.Header().Add("Content-Type", "application/x-protobuf")
		w.Write(b)
	}
}

func SendProtoJSON(w http.ResponseWriter, m protoreflect.ProtoMessage) {
	//w.Header().Add("Access-Control-Allow-Origin", "*")

	b, err := protojson.Marshal(m)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}
