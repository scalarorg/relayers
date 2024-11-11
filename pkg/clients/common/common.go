package common

type CommonConfig struct {
}

func NewCommonConfig() *CommonConfig {
	return &CommonConfig{}
}

// CallContract event contains chainId only,
// This function is used to get chain name from chainId based on the chains' config
func (c *CommonConfig) GetChainNameById(chainId uint64) (string, error) {
	return "", nil
}
