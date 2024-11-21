package db

type RelayDataStatus int

const (
	PENDING RelayDataStatus = iota
	APPROVED
	SUCCESS
	FAILED
)

type QueryOptions struct {
	IncludeCallContract          *bool
	IncludeCallContractWithToken *bool
}

type RelaydataExecuteResult struct {
	RelayDataId string
	Status      RelayDataStatus
}
