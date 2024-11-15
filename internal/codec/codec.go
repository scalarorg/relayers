package codec

import (
	//Importing the types package is necessary to register the codec
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"

	//Importing the secp256k1 package is necessary to register the type /cosmos.crypto.secp256k1.PubKey for the codec
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	_ "github.com/cosmos/cosmos-sdk/crypto/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/encoding"
	encproto "google.golang.org/grpc/encoding/proto"
)

var protoCodec *codec.ProtoCodec
var interfaceRegistry types.InterfaceRegistry

// This registers a codec that can encode custom Golang types defined by gogoproto extensions, which newer versions of the grpc module cannot.
// The fix has been extracted into its own module in order to minimize the number of dependencies
// that get imported before this init() function is called.
func init() {
	interfaceRegistry = types.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	//Register the secp256k1 pubkey type
	interfaceRegistry.RegisterImplementations((*proto.Message)(nil), &secp256k1.PubKey{})
	gogoCodec := GogoEnabled{Codec: encoding.GetCodec(encproto.Name)}
	fmt.Println(gogoCodec.Name())
	encoding.RegisterCodec(GogoEnabled{Codec: encoding.GetCodec(encproto.Name)})
	protoCodec = codec.NewProtoCodec(interfaceRegistry)
}

func GetInterfaceRegistry() types.InterfaceRegistry {
	return interfaceRegistry
}

func GetProtoCodec() *codec.ProtoCodec {
	return protoCodec
}

type GogoEnabled struct {
	encoding.Codec
}

func (c GogoEnabled) Marshal(v interface{}) ([]byte, error) {
	if vv, ok := v.(proto.Marshaler); ok {
		return vv.Marshal()
	}
	return c.Codec.Marshal(v)
}

func (c GogoEnabled) Unmarshal(data []byte, v interface{}) error {
	if vv, ok := v.(proto.Unmarshaler); ok {
		return vv.Unmarshal(data)
	}
	return c.Codec.Unmarshal(data, v)
}
