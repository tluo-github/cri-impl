package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/tluo-github/cri-impl/server"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

func Connect() (server.CriClient, *grpc.ClientConn) {
	conn, err := grpc.Dial("unix://"+OptHost, grpc.WithInsecure())
	if err != nil {
		klog.Fatalf("Connect with err:%v", err)
	}
	return server.NewCriClient(conn), conn
}

func Print(v interface{}) {
	fmt.Println(toString(v))
}

func toString(v interface{}) string {
	switch i := v.(type) {
	case proto.Message:
		s, err := (&jsonpb.Marshaler{EmitDefaults: true}).MarshalToString(i)
		if err != nil {
			klog.Fatalf("jsonpb marshal with err:%v", err)
		}
		return s
	default:
		s, err := json.Marshal(i)
		if err != nil {
			klog.Fatalf("json marshal with err:%v", err)
		}
		return string(s)
	}
}
