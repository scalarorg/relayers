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
