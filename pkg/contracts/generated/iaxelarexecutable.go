// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// IAxelarExecutableMetaData contains all meta data concerning the IAxelarExecutable contract.
var IAxelarExecutableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"NotApprovedByGateway\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"}],\"name\":\"execute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"tokenSymbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"executeWithToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gateway\",\"outputs\":[{\"internalType\":\"contractIAxelarGateway\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IAxelarExecutableABI is the input ABI used to generate the binding from.
// Deprecated: Use IAxelarExecutableMetaData.ABI instead.
var IAxelarExecutableABI = IAxelarExecutableMetaData.ABI

// IAxelarExecutable is an auto generated Go binding around an Ethereum contract.
type IAxelarExecutable struct {
	IAxelarExecutableCaller     // Read-only binding to the contract
	IAxelarExecutableTransactor // Write-only binding to the contract
	IAxelarExecutableFilterer   // Log filterer for contract events
}

// IAxelarExecutableCaller is an auto generated read-only Go binding around an Ethereum contract.
type IAxelarExecutableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarExecutableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IAxelarExecutableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarExecutableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IAxelarExecutableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarExecutableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IAxelarExecutableSession struct {
	Contract     *IAxelarExecutable // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IAxelarExecutableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IAxelarExecutableCallerSession struct {
	Contract *IAxelarExecutableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// IAxelarExecutableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IAxelarExecutableTransactorSession struct {
	Contract     *IAxelarExecutableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// IAxelarExecutableRaw is an auto generated low-level Go binding around an Ethereum contract.
type IAxelarExecutableRaw struct {
	Contract *IAxelarExecutable // Generic contract binding to access the raw methods on
}

// IAxelarExecutableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IAxelarExecutableCallerRaw struct {
	Contract *IAxelarExecutableCaller // Generic read-only contract binding to access the raw methods on
}

// IAxelarExecutableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IAxelarExecutableTransactorRaw struct {
	Contract *IAxelarExecutableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIAxelarExecutable creates a new instance of IAxelarExecutable, bound to a specific deployed contract.
func NewIAxelarExecutable(address common.Address, backend bind.ContractBackend) (*IAxelarExecutable, error) {
	contract, err := bindIAxelarExecutable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IAxelarExecutable{IAxelarExecutableCaller: IAxelarExecutableCaller{contract: contract}, IAxelarExecutableTransactor: IAxelarExecutableTransactor{contract: contract}, IAxelarExecutableFilterer: IAxelarExecutableFilterer{contract: contract}}, nil
}

// NewIAxelarExecutableCaller creates a new read-only instance of IAxelarExecutable, bound to a specific deployed contract.
func NewIAxelarExecutableCaller(address common.Address, caller bind.ContractCaller) (*IAxelarExecutableCaller, error) {
	contract, err := bindIAxelarExecutable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IAxelarExecutableCaller{contract: contract}, nil
}

// NewIAxelarExecutableTransactor creates a new write-only instance of IAxelarExecutable, bound to a specific deployed contract.
func NewIAxelarExecutableTransactor(address common.Address, transactor bind.ContractTransactor) (*IAxelarExecutableTransactor, error) {
	contract, err := bindIAxelarExecutable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IAxelarExecutableTransactor{contract: contract}, nil
}

// NewIAxelarExecutableFilterer creates a new log filterer instance of IAxelarExecutable, bound to a specific deployed contract.
func NewIAxelarExecutableFilterer(address common.Address, filterer bind.ContractFilterer) (*IAxelarExecutableFilterer, error) {
	contract, err := bindIAxelarExecutable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IAxelarExecutableFilterer{contract: contract}, nil
}

// bindIAxelarExecutable binds a generic wrapper to an already deployed contract.
func bindIAxelarExecutable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IAxelarExecutableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAxelarExecutable *IAxelarExecutableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAxelarExecutable.Contract.IAxelarExecutableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAxelarExecutable *IAxelarExecutableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.IAxelarExecutableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAxelarExecutable *IAxelarExecutableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.IAxelarExecutableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAxelarExecutable *IAxelarExecutableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAxelarExecutable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAxelarExecutable *IAxelarExecutableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAxelarExecutable *IAxelarExecutableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.contract.Transact(opts, method, params...)
}

// Gateway is a free data retrieval call binding the contract method 0x116191b6.
//
// Solidity: function gateway() view returns(address)
func (_IAxelarExecutable *IAxelarExecutableCaller) Gateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IAxelarExecutable.contract.Call(opts, &out, "gateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Gateway is a free data retrieval call binding the contract method 0x116191b6.
//
// Solidity: function gateway() view returns(address)
func (_IAxelarExecutable *IAxelarExecutableSession) Gateway() (common.Address, error) {
	return _IAxelarExecutable.Contract.Gateway(&_IAxelarExecutable.CallOpts)
}

// Gateway is a free data retrieval call binding the contract method 0x116191b6.
//
// Solidity: function gateway() view returns(address)
func (_IAxelarExecutable *IAxelarExecutableCallerSession) Gateway() (common.Address, error) {
	return _IAxelarExecutable.Contract.Gateway(&_IAxelarExecutable.CallOpts)
}

// Execute is a paid mutator transaction binding the contract method 0x49160658.
//
// Solidity: function execute(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload) returns()
func (_IAxelarExecutable *IAxelarExecutableTransactor) Execute(opts *bind.TransactOpts, commandId [32]byte, sourceChain string, sourceAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarExecutable.contract.Transact(opts, "execute", commandId, sourceChain, sourceAddress, payload)
}

// Execute is a paid mutator transaction binding the contract method 0x49160658.
//
// Solidity: function execute(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload) returns()
func (_IAxelarExecutable *IAxelarExecutableSession) Execute(commandId [32]byte, sourceChain string, sourceAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.Execute(&_IAxelarExecutable.TransactOpts, commandId, sourceChain, sourceAddress, payload)
}

// Execute is a paid mutator transaction binding the contract method 0x49160658.
//
// Solidity: function execute(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload) returns()
func (_IAxelarExecutable *IAxelarExecutableTransactorSession) Execute(commandId [32]byte, sourceChain string, sourceAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.Execute(&_IAxelarExecutable.TransactOpts, commandId, sourceChain, sourceAddress, payload)
}

// ExecuteWithToken is a paid mutator transaction binding the contract method 0x1a98b2e0.
//
// Solidity: function executeWithToken(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload, string tokenSymbol, uint256 amount) returns()
func (_IAxelarExecutable *IAxelarExecutableTransactor) ExecuteWithToken(opts *bind.TransactOpts, commandId [32]byte, sourceChain string, sourceAddress string, payload []byte, tokenSymbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarExecutable.contract.Transact(opts, "executeWithToken", commandId, sourceChain, sourceAddress, payload, tokenSymbol, amount)
}

// ExecuteWithToken is a paid mutator transaction binding the contract method 0x1a98b2e0.
//
// Solidity: function executeWithToken(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload, string tokenSymbol, uint256 amount) returns()
func (_IAxelarExecutable *IAxelarExecutableSession) ExecuteWithToken(commandId [32]byte, sourceChain string, sourceAddress string, payload []byte, tokenSymbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.ExecuteWithToken(&_IAxelarExecutable.TransactOpts, commandId, sourceChain, sourceAddress, payload, tokenSymbol, amount)
}

// ExecuteWithToken is a paid mutator transaction binding the contract method 0x1a98b2e0.
//
// Solidity: function executeWithToken(bytes32 commandId, string sourceChain, string sourceAddress, bytes payload, string tokenSymbol, uint256 amount) returns()
func (_IAxelarExecutable *IAxelarExecutableTransactorSession) ExecuteWithToken(commandId [32]byte, sourceChain string, sourceAddress string, payload []byte, tokenSymbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarExecutable.Contract.ExecuteWithToken(&_IAxelarExecutable.TransactOpts, commandId, sourceChain, sourceAddress, payload, tokenSymbol, amount)
}
