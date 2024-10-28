package evm

type Client struct {
    rpcURL string
}

func NewClient(rpcURL string) *Client {
    return &Client{rpcURL: rpcURL}
}

// Implement EVMClient interface methods
