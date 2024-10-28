package axelar

var AxelarService *Service

type Service struct {
	listener *AxelarListener
	client   *Client
}

func InitAxelarService() error {
	listener, err := NewAxelarListener()
	if err != nil {
		return err
	}
	signer, err := NewSignerClient()
	if err != nil {
		return err
	}
	client, err := NewClient(signer)
	if err != nil {
		return err
	}
	AxelarService = &Service{
		listener: listener,
		client:   client,
	}
	return nil
}
