package types

type Status int

const (
	PENDING Status = iota
	APPROVED
	SUCCESS
	FAILED
)
