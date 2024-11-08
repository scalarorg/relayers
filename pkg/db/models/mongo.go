package models

type ChainConfig struct {
	RpcUrl      string `bson:"rpc_url"`
	AccessToken string `bson:"access_token"`
}
