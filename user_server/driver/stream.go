package driver

import (
	"io"
	"net/http"

	streaming "github.com/cloudwego/kitex/pkg/streaming"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type HttpStream struct {
	r *http.Request
	w http.ResponseWriter
	streaming.Stream
}

func NewHttpStream(w http.ResponseWriter, r *http.Request) streaming.Stream {
	return &HttpStream{r: r, w: w}
}

func (x *HttpStream) RecvMsg(req interface{}) error {
	b, err := io.ReadAll(x.r.Body)
	if err != nil {
		return err
	}
	m, _ := req.(proto.Message)
	if err := protojson.Unmarshal(b, m); err != nil {
		return err
	}
	return nil
}

func (x *HttpStream) SendMsg(response interface{}) error {
	m, _ := response.(proto.Message)
	b, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(m)
	if err != nil {
		return err
	}
	x.w.Header().Set("Access-Control-Allow-Origin", "*")
	x.w.Header().Set("Access-Control-Allow-Headers", "*")
	x.w.Header().Set("Access-Control-Allow-Methods", "*")
	x.w.Header().Set("Access-Control-Expose-Headers", "*")
	x.w.Header().Set("Access-Control-Allow-Credentials", "true")
	if _, err := x.w.Write(b); err != nil {
		return err
	}
	return nil
}
