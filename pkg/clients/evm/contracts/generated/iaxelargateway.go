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

// IAxelarGatewayMetaData contains all meta data concerning the IAxelarGateway contract.
var IAxelarGatewayMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"BurnFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"ExceedMintLimit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAuthModule\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidChainId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidCodeHash\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidCommands\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSetMintLimitsParams\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidTokenDeployer\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"MintFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotProxy\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotSelf\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SetupFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"TokenAlreadyExists\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"TokenContractDoesNotExist\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"TokenDeployFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"TokenDoesNotExist\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationContractAddress\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"}],\"name\":\"ContractCall\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"sourceTxHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sourceEventIndex\",\"type\":\"uint256\"}],\"name\":\"ContractCallApproved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"sourceTxHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sourceEventIndex\",\"type\":\"uint256\"}],\"name\":\"ContractCallApprovedWithMint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationContractAddress\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"ContractCallWithToken\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"}],\"name\":\"Executed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"newOperatorsData\",\"type\":\"bytes\"}],\"name\":\"OperatorshipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"tokenAddresses\",\"type\":\"address\"}],\"name\":\"TokenDeployed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"limit\",\"type\":\"uint256\"}],\"name\":\"TokenMintLimitUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"destinationAddress\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TokenSent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"adminEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"name\":\"adminThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"name\":\"admins\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"allTokensFrozen\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"authModule\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"contractAddress\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"}],\"name\":\"callContract\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"contractAddress\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"callContractWithToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"input\",\"type\":\"bytes\"}],\"name\":\"execute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"implementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"}],\"name\":\"isCommandExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"isContractCallAndMintApproved\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"}],\"name\":\"isContractCallApproved\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"destinationChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"destinationAddress\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"sendToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string[]\",\"name\":\"symbols\",\"type\":\"string[]\"},{\"internalType\":\"uint256[]\",\"name\":\"limits\",\"type\":\"uint256[]\"}],\"name\":\"setTokenMintLimits\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"params\",\"type\":\"bytes\"}],\"name\":\"setup\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"tokenAddresses\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokenDeployer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"tokenFrozen\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"tokenMintAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"name\":\"tokenMintLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"newImplementationCodeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"setupParams\",\"type\":\"bytes\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"}],\"name\":\"validateContractCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commandId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"sourceChain\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"sourceAddress\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"validateContractCallAndMint\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IAxelarGatewayABI is the input ABI used to generate the binding from.
// Deprecated: Use IAxelarGatewayMetaData.ABI instead.
var IAxelarGatewayABI = IAxelarGatewayMetaData.ABI

// IAxelarGateway is an auto generated Go binding around an Ethereum contract.
type IAxelarGateway struct {
	IAxelarGatewayCaller     // Read-only binding to the contract
	IAxelarGatewayTransactor // Write-only binding to the contract
	IAxelarGatewayFilterer   // Log filterer for contract events
}

// IAxelarGatewayCaller is an auto generated read-only Go binding around an Ethereum contract.
type IAxelarGatewayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarGatewayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IAxelarGatewayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarGatewayFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IAxelarGatewayFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAxelarGatewaySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IAxelarGatewaySession struct {
	Contract     *IAxelarGateway   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IAxelarGatewayCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IAxelarGatewayCallerSession struct {
	Contract *IAxelarGatewayCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// IAxelarGatewayTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IAxelarGatewayTransactorSession struct {
	Contract     *IAxelarGatewayTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// IAxelarGatewayRaw is an auto generated low-level Go binding around an Ethereum contract.
type IAxelarGatewayRaw struct {
	Contract *IAxelarGateway // Generic contract binding to access the raw methods on
}

// IAxelarGatewayCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IAxelarGatewayCallerRaw struct {
	Contract *IAxelarGatewayCaller // Generic read-only contract binding to access the raw methods on
}

// IAxelarGatewayTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IAxelarGatewayTransactorRaw struct {
	Contract *IAxelarGatewayTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIAxelarGateway creates a new instance of IAxelarGateway, bound to a specific deployed contract.
func NewIAxelarGateway(address common.Address, backend bind.ContractBackend) (*IAxelarGateway, error) {
	contract, err := bindIAxelarGateway(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IAxelarGateway{IAxelarGatewayCaller: IAxelarGatewayCaller{contract: contract}, IAxelarGatewayTransactor: IAxelarGatewayTransactor{contract: contract}, IAxelarGatewayFilterer: IAxelarGatewayFilterer{contract: contract}}, nil
}

// NewIAxelarGatewayCaller creates a new read-only instance of IAxelarGateway, bound to a specific deployed contract.
func NewIAxelarGatewayCaller(address common.Address, caller bind.ContractCaller) (*IAxelarGatewayCaller, error) {
	contract, err := bindIAxelarGateway(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayCaller{contract: contract}, nil
}

// NewIAxelarGatewayTransactor creates a new write-only instance of IAxelarGateway, bound to a specific deployed contract.
func NewIAxelarGatewayTransactor(address common.Address, transactor bind.ContractTransactor) (*IAxelarGatewayTransactor, error) {
	contract, err := bindIAxelarGateway(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayTransactor{contract: contract}, nil
}

// NewIAxelarGatewayFilterer creates a new log filterer instance of IAxelarGateway, bound to a specific deployed contract.
func NewIAxelarGatewayFilterer(address common.Address, filterer bind.ContractFilterer) (*IAxelarGatewayFilterer, error) {
	contract, err := bindIAxelarGateway(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayFilterer{contract: contract}, nil
}

// bindIAxelarGateway binds a generic wrapper to an already deployed contract.
func bindIAxelarGateway(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IAxelarGatewayMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAxelarGateway *IAxelarGatewayRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAxelarGateway.Contract.IAxelarGatewayCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAxelarGateway *IAxelarGatewayRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.IAxelarGatewayTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAxelarGateway *IAxelarGatewayRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.IAxelarGatewayTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAxelarGateway *IAxelarGatewayCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAxelarGateway.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAxelarGateway *IAxelarGatewayTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAxelarGateway *IAxelarGatewayTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.contract.Transact(opts, method, params...)
}

// AdminEpoch is a free data retrieval call binding the contract method 0x364940d8.
//
// Solidity: function adminEpoch() view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCaller) AdminEpoch(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "adminEpoch")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AdminEpoch is a free data retrieval call binding the contract method 0x364940d8.
//
// Solidity: function adminEpoch() view returns(uint256)
func (_IAxelarGateway *IAxelarGatewaySession) AdminEpoch() (*big.Int, error) {
	return _IAxelarGateway.Contract.AdminEpoch(&_IAxelarGateway.CallOpts)
}

// AdminEpoch is a free data retrieval call binding the contract method 0x364940d8.
//
// Solidity: function adminEpoch() view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCallerSession) AdminEpoch() (*big.Int, error) {
	return _IAxelarGateway.Contract.AdminEpoch(&_IAxelarGateway.CallOpts)
}

// AdminThreshold is a free data retrieval call binding the contract method 0x88b30587.
//
// Solidity: function adminThreshold(uint256 epoch) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCaller) AdminThreshold(opts *bind.CallOpts, epoch *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "adminThreshold", epoch)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AdminThreshold is a free data retrieval call binding the contract method 0x88b30587.
//
// Solidity: function adminThreshold(uint256 epoch) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewaySession) AdminThreshold(epoch *big.Int) (*big.Int, error) {
	return _IAxelarGateway.Contract.AdminThreshold(&_IAxelarGateway.CallOpts, epoch)
}

// AdminThreshold is a free data retrieval call binding the contract method 0x88b30587.
//
// Solidity: function adminThreshold(uint256 epoch) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCallerSession) AdminThreshold(epoch *big.Int) (*big.Int, error) {
	return _IAxelarGateway.Contract.AdminThreshold(&_IAxelarGateway.CallOpts, epoch)
}

// Admins is a free data retrieval call binding the contract method 0x14bfd6d0.
//
// Solidity: function admins(uint256 epoch) view returns(address[])
func (_IAxelarGateway *IAxelarGatewayCaller) Admins(opts *bind.CallOpts, epoch *big.Int) ([]common.Address, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "admins", epoch)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// Admins is a free data retrieval call binding the contract method 0x14bfd6d0.
//
// Solidity: function admins(uint256 epoch) view returns(address[])
func (_IAxelarGateway *IAxelarGatewaySession) Admins(epoch *big.Int) ([]common.Address, error) {
	return _IAxelarGateway.Contract.Admins(&_IAxelarGateway.CallOpts, epoch)
}

// Admins is a free data retrieval call binding the contract method 0x14bfd6d0.
//
// Solidity: function admins(uint256 epoch) view returns(address[])
func (_IAxelarGateway *IAxelarGatewayCallerSession) Admins(epoch *big.Int) ([]common.Address, error) {
	return _IAxelarGateway.Contract.Admins(&_IAxelarGateway.CallOpts, epoch)
}

// AllTokensFrozen is a free data retrieval call binding the contract method 0xaa1e1f0a.
//
// Solidity: function allTokensFrozen() view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCaller) AllTokensFrozen(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "allTokensFrozen")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllTokensFrozen is a free data retrieval call binding the contract method 0xaa1e1f0a.
//
// Solidity: function allTokensFrozen() view returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) AllTokensFrozen() (bool, error) {
	return _IAxelarGateway.Contract.AllTokensFrozen(&_IAxelarGateway.CallOpts)
}

// AllTokensFrozen is a free data retrieval call binding the contract method 0xaa1e1f0a.
//
// Solidity: function allTokensFrozen() view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCallerSession) AllTokensFrozen() (bool, error) {
	return _IAxelarGateway.Contract.AllTokensFrozen(&_IAxelarGateway.CallOpts)
}

// AuthModule is a free data retrieval call binding the contract method 0x64940c56.
//
// Solidity: function authModule() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCaller) AuthModule(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "authModule")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AuthModule is a free data retrieval call binding the contract method 0x64940c56.
//
// Solidity: function authModule() view returns(address)
func (_IAxelarGateway *IAxelarGatewaySession) AuthModule() (common.Address, error) {
	return _IAxelarGateway.Contract.AuthModule(&_IAxelarGateway.CallOpts)
}

// AuthModule is a free data retrieval call binding the contract method 0x64940c56.
//
// Solidity: function authModule() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCallerSession) AuthModule() (common.Address, error) {
	return _IAxelarGateway.Contract.AuthModule(&_IAxelarGateway.CallOpts)
}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCaller) Implementation(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "implementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_IAxelarGateway *IAxelarGatewaySession) Implementation() (common.Address, error) {
	return _IAxelarGateway.Contract.Implementation(&_IAxelarGateway.CallOpts)
}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCallerSession) Implementation() (common.Address, error) {
	return _IAxelarGateway.Contract.Implementation(&_IAxelarGateway.CallOpts)
}

// IsCommandExecuted is a free data retrieval call binding the contract method 0xd26ff210.
//
// Solidity: function isCommandExecuted(bytes32 commandId) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCaller) IsCommandExecuted(opts *bind.CallOpts, commandId [32]byte) (bool, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "isCommandExecuted", commandId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCommandExecuted is a free data retrieval call binding the contract method 0xd26ff210.
//
// Solidity: function isCommandExecuted(bytes32 commandId) view returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) IsCommandExecuted(commandId [32]byte) (bool, error) {
	return _IAxelarGateway.Contract.IsCommandExecuted(&_IAxelarGateway.CallOpts, commandId)
}

// IsCommandExecuted is a free data retrieval call binding the contract method 0xd26ff210.
//
// Solidity: function isCommandExecuted(bytes32 commandId) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCallerSession) IsCommandExecuted(commandId [32]byte) (bool, error) {
	return _IAxelarGateway.Contract.IsCommandExecuted(&_IAxelarGateway.CallOpts, commandId)
}

// IsContractCallAndMintApproved is a free data retrieval call binding the contract method 0xbc00c216.
//
// Solidity: function isContractCallAndMintApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash, string symbol, uint256 amount) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCaller) IsContractCallAndMintApproved(opts *bind.CallOpts, commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte, symbol string, amount *big.Int) (bool, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "isContractCallAndMintApproved", commandId, sourceChain, sourceAddress, contractAddress, payloadHash, symbol, amount)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsContractCallAndMintApproved is a free data retrieval call binding the contract method 0xbc00c216.
//
// Solidity: function isContractCallAndMintApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash, string symbol, uint256 amount) view returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) IsContractCallAndMintApproved(commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte, symbol string, amount *big.Int) (bool, error) {
	return _IAxelarGateway.Contract.IsContractCallAndMintApproved(&_IAxelarGateway.CallOpts, commandId, sourceChain, sourceAddress, contractAddress, payloadHash, symbol, amount)
}

// IsContractCallAndMintApproved is a free data retrieval call binding the contract method 0xbc00c216.
//
// Solidity: function isContractCallAndMintApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash, string symbol, uint256 amount) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCallerSession) IsContractCallAndMintApproved(commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte, symbol string, amount *big.Int) (bool, error) {
	return _IAxelarGateway.Contract.IsContractCallAndMintApproved(&_IAxelarGateway.CallOpts, commandId, sourceChain, sourceAddress, contractAddress, payloadHash, symbol, amount)
}

// IsContractCallApproved is a free data retrieval call binding the contract method 0xf6a5f9f5.
//
// Solidity: function isContractCallApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCaller) IsContractCallApproved(opts *bind.CallOpts, commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte) (bool, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "isContractCallApproved", commandId, sourceChain, sourceAddress, contractAddress, payloadHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsContractCallApproved is a free data retrieval call binding the contract method 0xf6a5f9f5.
//
// Solidity: function isContractCallApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash) view returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) IsContractCallApproved(commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte) (bool, error) {
	return _IAxelarGateway.Contract.IsContractCallApproved(&_IAxelarGateway.CallOpts, commandId, sourceChain, sourceAddress, contractAddress, payloadHash)
}

// IsContractCallApproved is a free data retrieval call binding the contract method 0xf6a5f9f5.
//
// Solidity: function isContractCallApproved(bytes32 commandId, string sourceChain, string sourceAddress, address contractAddress, bytes32 payloadHash) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCallerSession) IsContractCallApproved(commandId [32]byte, sourceChain string, sourceAddress string, contractAddress common.Address, payloadHash [32]byte) (bool, error) {
	return _IAxelarGateway.Contract.IsContractCallApproved(&_IAxelarGateway.CallOpts, commandId, sourceChain, sourceAddress, contractAddress, payloadHash)
}

// TokenAddresses is a free data retrieval call binding the contract method 0x935b13f6.
//
// Solidity: function tokenAddresses(string symbol) view returns(address)
func (_IAxelarGateway *IAxelarGatewayCaller) TokenAddresses(opts *bind.CallOpts, symbol string) (common.Address, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "tokenAddresses", symbol)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenAddresses is a free data retrieval call binding the contract method 0x935b13f6.
//
// Solidity: function tokenAddresses(string symbol) view returns(address)
func (_IAxelarGateway *IAxelarGatewaySession) TokenAddresses(symbol string) (common.Address, error) {
	return _IAxelarGateway.Contract.TokenAddresses(&_IAxelarGateway.CallOpts, symbol)
}

// TokenAddresses is a free data retrieval call binding the contract method 0x935b13f6.
//
// Solidity: function tokenAddresses(string symbol) view returns(address)
func (_IAxelarGateway *IAxelarGatewayCallerSession) TokenAddresses(symbol string) (common.Address, error) {
	return _IAxelarGateway.Contract.TokenAddresses(&_IAxelarGateway.CallOpts, symbol)
}

// TokenDeployer is a free data retrieval call binding the contract method 0x2a2dae0a.
//
// Solidity: function tokenDeployer() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCaller) TokenDeployer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "tokenDeployer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenDeployer is a free data retrieval call binding the contract method 0x2a2dae0a.
//
// Solidity: function tokenDeployer() view returns(address)
func (_IAxelarGateway *IAxelarGatewaySession) TokenDeployer() (common.Address, error) {
	return _IAxelarGateway.Contract.TokenDeployer(&_IAxelarGateway.CallOpts)
}

// TokenDeployer is a free data retrieval call binding the contract method 0x2a2dae0a.
//
// Solidity: function tokenDeployer() view returns(address)
func (_IAxelarGateway *IAxelarGatewayCallerSession) TokenDeployer() (common.Address, error) {
	return _IAxelarGateway.Contract.TokenDeployer(&_IAxelarGateway.CallOpts)
}

// TokenFrozen is a free data retrieval call binding the contract method 0x7b1b769e.
//
// Solidity: function tokenFrozen(string symbol) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCaller) TokenFrozen(opts *bind.CallOpts, symbol string) (bool, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "tokenFrozen", symbol)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TokenFrozen is a free data retrieval call binding the contract method 0x7b1b769e.
//
// Solidity: function tokenFrozen(string symbol) view returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) TokenFrozen(symbol string) (bool, error) {
	return _IAxelarGateway.Contract.TokenFrozen(&_IAxelarGateway.CallOpts, symbol)
}

// TokenFrozen is a free data retrieval call binding the contract method 0x7b1b769e.
//
// Solidity: function tokenFrozen(string symbol) view returns(bool)
func (_IAxelarGateway *IAxelarGatewayCallerSession) TokenFrozen(symbol string) (bool, error) {
	return _IAxelarGateway.Contract.TokenFrozen(&_IAxelarGateway.CallOpts, symbol)
}

// TokenMintAmount is a free data retrieval call binding the contract method 0xcec7b359.
//
// Solidity: function tokenMintAmount(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCaller) TokenMintAmount(opts *bind.CallOpts, symbol string) (*big.Int, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "tokenMintAmount", symbol)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenMintAmount is a free data retrieval call binding the contract method 0xcec7b359.
//
// Solidity: function tokenMintAmount(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewaySession) TokenMintAmount(symbol string) (*big.Int, error) {
	return _IAxelarGateway.Contract.TokenMintAmount(&_IAxelarGateway.CallOpts, symbol)
}

// TokenMintAmount is a free data retrieval call binding the contract method 0xcec7b359.
//
// Solidity: function tokenMintAmount(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCallerSession) TokenMintAmount(symbol string) (*big.Int, error) {
	return _IAxelarGateway.Contract.TokenMintAmount(&_IAxelarGateway.CallOpts, symbol)
}

// TokenMintLimit is a free data retrieval call binding the contract method 0x269eb65e.
//
// Solidity: function tokenMintLimit(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCaller) TokenMintLimit(opts *bind.CallOpts, symbol string) (*big.Int, error) {
	var out []interface{}
	err := _IAxelarGateway.contract.Call(opts, &out, "tokenMintLimit", symbol)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenMintLimit is a free data retrieval call binding the contract method 0x269eb65e.
//
// Solidity: function tokenMintLimit(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewaySession) TokenMintLimit(symbol string) (*big.Int, error) {
	return _IAxelarGateway.Contract.TokenMintLimit(&_IAxelarGateway.CallOpts, symbol)
}

// TokenMintLimit is a free data retrieval call binding the contract method 0x269eb65e.
//
// Solidity: function tokenMintLimit(string symbol) view returns(uint256)
func (_IAxelarGateway *IAxelarGatewayCallerSession) TokenMintLimit(symbol string) (*big.Int, error) {
	return _IAxelarGateway.Contract.TokenMintLimit(&_IAxelarGateway.CallOpts, symbol)
}

// CallContract is a paid mutator transaction binding the contract method 0x1c92115f.
//
// Solidity: function callContract(string destinationChain, string contractAddress, bytes payload) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) CallContract(opts *bind.TransactOpts, destinationChain string, contractAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "callContract", destinationChain, contractAddress, payload)
}

// CallContract is a paid mutator transaction binding the contract method 0x1c92115f.
//
// Solidity: function callContract(string destinationChain, string contractAddress, bytes payload) returns()
func (_IAxelarGateway *IAxelarGatewaySession) CallContract(destinationChain string, contractAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.CallContract(&_IAxelarGateway.TransactOpts, destinationChain, contractAddress, payload)
}

// CallContract is a paid mutator transaction binding the contract method 0x1c92115f.
//
// Solidity: function callContract(string destinationChain, string contractAddress, bytes payload) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) CallContract(destinationChain string, contractAddress string, payload []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.CallContract(&_IAxelarGateway.TransactOpts, destinationChain, contractAddress, payload)
}

// CallContractWithToken is a paid mutator transaction binding the contract method 0xb5417084.
//
// Solidity: function callContractWithToken(string destinationChain, string contractAddress, bytes payload, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) CallContractWithToken(opts *bind.TransactOpts, destinationChain string, contractAddress string, payload []byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "callContractWithToken", destinationChain, contractAddress, payload, symbol, amount)
}

// CallContractWithToken is a paid mutator transaction binding the contract method 0xb5417084.
//
// Solidity: function callContractWithToken(string destinationChain, string contractAddress, bytes payload, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewaySession) CallContractWithToken(destinationChain string, contractAddress string, payload []byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.CallContractWithToken(&_IAxelarGateway.TransactOpts, destinationChain, contractAddress, payload, symbol, amount)
}

// CallContractWithToken is a paid mutator transaction binding the contract method 0xb5417084.
//
// Solidity: function callContractWithToken(string destinationChain, string contractAddress, bytes payload, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) CallContractWithToken(destinationChain string, contractAddress string, payload []byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.CallContractWithToken(&_IAxelarGateway.TransactOpts, destinationChain, contractAddress, payload, symbol, amount)
}

// Execute is a paid mutator transaction binding the contract method 0x09c5eabe.
//
// Solidity: function execute(bytes input) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) Execute(opts *bind.TransactOpts, input []byte) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "execute", input)
}

// Execute is a paid mutator transaction binding the contract method 0x09c5eabe.
//
// Solidity: function execute(bytes input) returns()
func (_IAxelarGateway *IAxelarGatewaySession) Execute(input []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Execute(&_IAxelarGateway.TransactOpts, input)
}

// Execute is a paid mutator transaction binding the contract method 0x09c5eabe.
//
// Solidity: function execute(bytes input) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) Execute(input []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Execute(&_IAxelarGateway.TransactOpts, input)
}

// SendToken is a paid mutator transaction binding the contract method 0x26ef699d.
//
// Solidity: function sendToken(string destinationChain, string destinationAddress, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) SendToken(opts *bind.TransactOpts, destinationChain string, destinationAddress string, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "sendToken", destinationChain, destinationAddress, symbol, amount)
}

// SendToken is a paid mutator transaction binding the contract method 0x26ef699d.
//
// Solidity: function sendToken(string destinationChain, string destinationAddress, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewaySession) SendToken(destinationChain string, destinationAddress string, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.SendToken(&_IAxelarGateway.TransactOpts, destinationChain, destinationAddress, symbol, amount)
}

// SendToken is a paid mutator transaction binding the contract method 0x26ef699d.
//
// Solidity: function sendToken(string destinationChain, string destinationAddress, string symbol, uint256 amount) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) SendToken(destinationChain string, destinationAddress string, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.SendToken(&_IAxelarGateway.TransactOpts, destinationChain, destinationAddress, symbol, amount)
}

// SetTokenMintLimits is a paid mutator transaction binding the contract method 0x67ace8eb.
//
// Solidity: function setTokenMintLimits(string[] symbols, uint256[] limits) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) SetTokenMintLimits(opts *bind.TransactOpts, symbols []string, limits []*big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "setTokenMintLimits", symbols, limits)
}

// SetTokenMintLimits is a paid mutator transaction binding the contract method 0x67ace8eb.
//
// Solidity: function setTokenMintLimits(string[] symbols, uint256[] limits) returns()
func (_IAxelarGateway *IAxelarGatewaySession) SetTokenMintLimits(symbols []string, limits []*big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.SetTokenMintLimits(&_IAxelarGateway.TransactOpts, symbols, limits)
}

// SetTokenMintLimits is a paid mutator transaction binding the contract method 0x67ace8eb.
//
// Solidity: function setTokenMintLimits(string[] symbols, uint256[] limits) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) SetTokenMintLimits(symbols []string, limits []*big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.SetTokenMintLimits(&_IAxelarGateway.TransactOpts, symbols, limits)
}

// Setup is a paid mutator transaction binding the contract method 0x9ded06df.
//
// Solidity: function setup(bytes params) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) Setup(opts *bind.TransactOpts, params []byte) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "setup", params)
}

// Setup is a paid mutator transaction binding the contract method 0x9ded06df.
//
// Solidity: function setup(bytes params) returns()
func (_IAxelarGateway *IAxelarGatewaySession) Setup(params []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Setup(&_IAxelarGateway.TransactOpts, params)
}

// Setup is a paid mutator transaction binding the contract method 0x9ded06df.
//
// Solidity: function setup(bytes params) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) Setup(params []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Setup(&_IAxelarGateway.TransactOpts, params)
}

// Upgrade is a paid mutator transaction binding the contract method 0xa3499c73.
//
// Solidity: function upgrade(address newImplementation, bytes32 newImplementationCodeHash, bytes setupParams) returns()
func (_IAxelarGateway *IAxelarGatewayTransactor) Upgrade(opts *bind.TransactOpts, newImplementation common.Address, newImplementationCodeHash [32]byte, setupParams []byte) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "upgrade", newImplementation, newImplementationCodeHash, setupParams)
}

// Upgrade is a paid mutator transaction binding the contract method 0xa3499c73.
//
// Solidity: function upgrade(address newImplementation, bytes32 newImplementationCodeHash, bytes setupParams) returns()
func (_IAxelarGateway *IAxelarGatewaySession) Upgrade(newImplementation common.Address, newImplementationCodeHash [32]byte, setupParams []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Upgrade(&_IAxelarGateway.TransactOpts, newImplementation, newImplementationCodeHash, setupParams)
}

// Upgrade is a paid mutator transaction binding the contract method 0xa3499c73.
//
// Solidity: function upgrade(address newImplementation, bytes32 newImplementationCodeHash, bytes setupParams) returns()
func (_IAxelarGateway *IAxelarGatewayTransactorSession) Upgrade(newImplementation common.Address, newImplementationCodeHash [32]byte, setupParams []byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.Upgrade(&_IAxelarGateway.TransactOpts, newImplementation, newImplementationCodeHash, setupParams)
}

// ValidateContractCall is a paid mutator transaction binding the contract method 0x5f6970c3.
//
// Solidity: function validateContractCall(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash) returns(bool)
func (_IAxelarGateway *IAxelarGatewayTransactor) ValidateContractCall(opts *bind.TransactOpts, commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "validateContractCall", commandId, sourceChain, sourceAddress, payloadHash)
}

// ValidateContractCall is a paid mutator transaction binding the contract method 0x5f6970c3.
//
// Solidity: function validateContractCall(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash) returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) ValidateContractCall(commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.ValidateContractCall(&_IAxelarGateway.TransactOpts, commandId, sourceChain, sourceAddress, payloadHash)
}

// ValidateContractCall is a paid mutator transaction binding the contract method 0x5f6970c3.
//
// Solidity: function validateContractCall(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash) returns(bool)
func (_IAxelarGateway *IAxelarGatewayTransactorSession) ValidateContractCall(commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.ValidateContractCall(&_IAxelarGateway.TransactOpts, commandId, sourceChain, sourceAddress, payloadHash)
}

// ValidateContractCallAndMint is a paid mutator transaction binding the contract method 0x1876eed9.
//
// Solidity: function validateContractCallAndMint(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash, string symbol, uint256 amount) returns(bool)
func (_IAxelarGateway *IAxelarGatewayTransactor) ValidateContractCallAndMint(opts *bind.TransactOpts, commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.contract.Transact(opts, "validateContractCallAndMint", commandId, sourceChain, sourceAddress, payloadHash, symbol, amount)
}

// ValidateContractCallAndMint is a paid mutator transaction binding the contract method 0x1876eed9.
//
// Solidity: function validateContractCallAndMint(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash, string symbol, uint256 amount) returns(bool)
func (_IAxelarGateway *IAxelarGatewaySession) ValidateContractCallAndMint(commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.ValidateContractCallAndMint(&_IAxelarGateway.TransactOpts, commandId, sourceChain, sourceAddress, payloadHash, symbol, amount)
}

// ValidateContractCallAndMint is a paid mutator transaction binding the contract method 0x1876eed9.
//
// Solidity: function validateContractCallAndMint(bytes32 commandId, string sourceChain, string sourceAddress, bytes32 payloadHash, string symbol, uint256 amount) returns(bool)
func (_IAxelarGateway *IAxelarGatewayTransactorSession) ValidateContractCallAndMint(commandId [32]byte, sourceChain string, sourceAddress string, payloadHash [32]byte, symbol string, amount *big.Int) (*types.Transaction, error) {
	return _IAxelarGateway.Contract.ValidateContractCallAndMint(&_IAxelarGateway.TransactOpts, commandId, sourceChain, sourceAddress, payloadHash, symbol, amount)
}

// IAxelarGatewayContractCallIterator is returned from FilterContractCall and is used to iterate over the raw logs and unpacked data for ContractCall events raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallIterator struct {
	Event *IAxelarGatewayContractCall // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayContractCallIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayContractCall)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayContractCall)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayContractCallIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayContractCallIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayContractCall represents a ContractCall event raised by the IAxelarGateway contract.
type IAxelarGatewayContractCall struct {
	Sender                     common.Address
	DestinationChain           string
	DestinationContractAddress string
	PayloadHash                [32]byte
	Payload                    []byte
	Raw                        types.Log // Blockchain specific contextual infos
}

// FilterContractCall is a free log retrieval operation binding the contract event 0x30ae6cc78c27e651745bf2ad08a11de83910ac1e347a52f7ac898c0fbef94dae.
//
// Solidity: event ContractCall(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterContractCall(opts *bind.FilterOpts, sender []common.Address, payloadHash [][32]byte) (*IAxelarGatewayContractCallIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "ContractCall", senderRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayContractCallIterator{contract: _IAxelarGateway.contract, event: "ContractCall", logs: logs, sub: sub}, nil
}

// WatchContractCall is a free log subscription operation binding the contract event 0x30ae6cc78c27e651745bf2ad08a11de83910ac1e347a52f7ac898c0fbef94dae.
//
// Solidity: event ContractCall(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchContractCall(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayContractCall, sender []common.Address, payloadHash [][32]byte) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "ContractCall", senderRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayContractCall)
				if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCall", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseContractCall is a log parse operation binding the contract event 0x30ae6cc78c27e651745bf2ad08a11de83910ac1e347a52f7ac898c0fbef94dae.
//
// Solidity: event ContractCall(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseContractCall(log types.Log) (*IAxelarGatewayContractCall, error) {
	event := new(IAxelarGatewayContractCall)
	if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCall", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayContractCallApprovedIterator is returned from FilterContractCallApproved and is used to iterate over the raw logs and unpacked data for ContractCallApproved events raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallApprovedIterator struct {
	Event *IAxelarGatewayContractCallApproved // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayContractCallApprovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayContractCallApproved)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayContractCallApproved)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayContractCallApprovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayContractCallApprovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayContractCallApproved represents a ContractCallApproved event raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallApproved struct {
	CommandId        [32]byte
	SourceChain      string
	SourceAddress    string
	ContractAddress  common.Address
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterContractCallApproved is a free log retrieval operation binding the contract event 0x44e4f8f6bd682c5a3aeba93601ab07cb4d1f21b2aab1ae4880d9577919309aa4.
//
// Solidity: event ContractCallApproved(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterContractCallApproved(opts *bind.FilterOpts, commandId [][32]byte, contractAddress []common.Address, payloadHash [][32]byte) (*IAxelarGatewayContractCallApprovedIterator, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	var contractAddressRule []interface{}
	for _, contractAddressItem := range contractAddress {
		contractAddressRule = append(contractAddressRule, contractAddressItem)
	}
	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "ContractCallApproved", commandIdRule, contractAddressRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayContractCallApprovedIterator{contract: _IAxelarGateway.contract, event: "ContractCallApproved", logs: logs, sub: sub}, nil
}

// WatchContractCallApproved is a free log subscription operation binding the contract event 0x44e4f8f6bd682c5a3aeba93601ab07cb4d1f21b2aab1ae4880d9577919309aa4.
//
// Solidity: event ContractCallApproved(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchContractCallApproved(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayContractCallApproved, commandId [][32]byte, contractAddress []common.Address, payloadHash [][32]byte) (event.Subscription, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	var contractAddressRule []interface{}
	for _, contractAddressItem := range contractAddress {
		contractAddressRule = append(contractAddressRule, contractAddressItem)
	}
	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "ContractCallApproved", commandIdRule, contractAddressRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayContractCallApproved)
				if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallApproved", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseContractCallApproved is a log parse operation binding the contract event 0x44e4f8f6bd682c5a3aeba93601ab07cb4d1f21b2aab1ae4880d9577919309aa4.
//
// Solidity: event ContractCallApproved(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseContractCallApproved(log types.Log) (*IAxelarGatewayContractCallApproved, error) {
	event := new(IAxelarGatewayContractCallApproved)
	if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallApproved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayContractCallApprovedWithMintIterator is returned from FilterContractCallApprovedWithMint and is used to iterate over the raw logs and unpacked data for ContractCallApprovedWithMint events raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallApprovedWithMintIterator struct {
	Event *IAxelarGatewayContractCallApprovedWithMint // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayContractCallApprovedWithMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayContractCallApprovedWithMint)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayContractCallApprovedWithMint)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayContractCallApprovedWithMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayContractCallApprovedWithMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayContractCallApprovedWithMint represents a ContractCallApprovedWithMint event raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallApprovedWithMint struct {
	CommandId        [32]byte
	SourceChain      string
	SourceAddress    string
	ContractAddress  common.Address
	PayloadHash      [32]byte
	Symbol           string
	Amount           *big.Int
	SourceTxHash     [32]byte
	SourceEventIndex *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterContractCallApprovedWithMint is a free log retrieval operation binding the contract event 0x9991faa1f435675159ffae64b66d7ecfdb55c29755869a18db8497b4392347e0.
//
// Solidity: event ContractCallApprovedWithMint(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, string symbol, uint256 amount, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterContractCallApprovedWithMint(opts *bind.FilterOpts, commandId [][32]byte, contractAddress []common.Address, payloadHash [][32]byte) (*IAxelarGatewayContractCallApprovedWithMintIterator, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	var contractAddressRule []interface{}
	for _, contractAddressItem := range contractAddress {
		contractAddressRule = append(contractAddressRule, contractAddressItem)
	}
	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "ContractCallApprovedWithMint", commandIdRule, contractAddressRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayContractCallApprovedWithMintIterator{contract: _IAxelarGateway.contract, event: "ContractCallApprovedWithMint", logs: logs, sub: sub}, nil
}

// WatchContractCallApprovedWithMint is a free log subscription operation binding the contract event 0x9991faa1f435675159ffae64b66d7ecfdb55c29755869a18db8497b4392347e0.
//
// Solidity: event ContractCallApprovedWithMint(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, string symbol, uint256 amount, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchContractCallApprovedWithMint(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayContractCallApprovedWithMint, commandId [][32]byte, contractAddress []common.Address, payloadHash [][32]byte) (event.Subscription, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	var contractAddressRule []interface{}
	for _, contractAddressItem := range contractAddress {
		contractAddressRule = append(contractAddressRule, contractAddressItem)
	}
	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "ContractCallApprovedWithMint", commandIdRule, contractAddressRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayContractCallApprovedWithMint)
				if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallApprovedWithMint", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseContractCallApprovedWithMint is a log parse operation binding the contract event 0x9991faa1f435675159ffae64b66d7ecfdb55c29755869a18db8497b4392347e0.
//
// Solidity: event ContractCallApprovedWithMint(bytes32 indexed commandId, string sourceChain, string sourceAddress, address indexed contractAddress, bytes32 indexed payloadHash, string symbol, uint256 amount, bytes32 sourceTxHash, uint256 sourceEventIndex)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseContractCallApprovedWithMint(log types.Log) (*IAxelarGatewayContractCallApprovedWithMint, error) {
	event := new(IAxelarGatewayContractCallApprovedWithMint)
	if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallApprovedWithMint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayContractCallWithTokenIterator is returned from FilterContractCallWithToken and is used to iterate over the raw logs and unpacked data for ContractCallWithToken events raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallWithTokenIterator struct {
	Event *IAxelarGatewayContractCallWithToken // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayContractCallWithTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayContractCallWithToken)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayContractCallWithToken)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayContractCallWithTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayContractCallWithTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayContractCallWithToken represents a ContractCallWithToken event raised by the IAxelarGateway contract.
type IAxelarGatewayContractCallWithToken struct {
	Sender                     common.Address
	DestinationChain           string
	DestinationContractAddress string
	PayloadHash                [32]byte
	Payload                    []byte
	Symbol                     string
	Amount                     *big.Int
	Raw                        types.Log // Blockchain specific contextual infos
}

// FilterContractCallWithToken is a free log retrieval operation binding the contract event 0x7e50569d26be643bda7757722291ec66b1be66d8283474ae3fab5a98f878a7a2.
//
// Solidity: event ContractCallWithToken(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterContractCallWithToken(opts *bind.FilterOpts, sender []common.Address, payloadHash [][32]byte) (*IAxelarGatewayContractCallWithTokenIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "ContractCallWithToken", senderRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayContractCallWithTokenIterator{contract: _IAxelarGateway.contract, event: "ContractCallWithToken", logs: logs, sub: sub}, nil
}

// WatchContractCallWithToken is a free log subscription operation binding the contract event 0x7e50569d26be643bda7757722291ec66b1be66d8283474ae3fab5a98f878a7a2.
//
// Solidity: event ContractCallWithToken(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchContractCallWithToken(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayContractCallWithToken, sender []common.Address, payloadHash [][32]byte) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var payloadHashRule []interface{}
	for _, payloadHashItem := range payloadHash {
		payloadHashRule = append(payloadHashRule, payloadHashItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "ContractCallWithToken", senderRule, payloadHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayContractCallWithToken)
				if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallWithToken", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseContractCallWithToken is a log parse operation binding the contract event 0x7e50569d26be643bda7757722291ec66b1be66d8283474ae3fab5a98f878a7a2.
//
// Solidity: event ContractCallWithToken(address indexed sender, string destinationChain, string destinationContractAddress, bytes32 indexed payloadHash, bytes payload, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseContractCallWithToken(log types.Log) (*IAxelarGatewayContractCallWithToken, error) {
	event := new(IAxelarGatewayContractCallWithToken)
	if err := _IAxelarGateway.contract.UnpackLog(event, "ContractCallWithToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayExecutedIterator is returned from FilterExecuted and is used to iterate over the raw logs and unpacked data for Executed events raised by the IAxelarGateway contract.
type IAxelarGatewayExecutedIterator struct {
	Event *IAxelarGatewayExecuted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayExecuted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayExecuted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayExecuted represents a Executed event raised by the IAxelarGateway contract.
type IAxelarGatewayExecuted struct {
	CommandId [32]byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterExecuted is a free log retrieval operation binding the contract event 0xa74c8847d513feba22a0f0cb38d53081abf97562cdb293926ba243689e7c41ca.
//
// Solidity: event Executed(bytes32 indexed commandId)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterExecuted(opts *bind.FilterOpts, commandId [][32]byte) (*IAxelarGatewayExecutedIterator, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "Executed", commandIdRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayExecutedIterator{contract: _IAxelarGateway.contract, event: "Executed", logs: logs, sub: sub}, nil
}

// WatchExecuted is a free log subscription operation binding the contract event 0xa74c8847d513feba22a0f0cb38d53081abf97562cdb293926ba243689e7c41ca.
//
// Solidity: event Executed(bytes32 indexed commandId)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchExecuted(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayExecuted, commandId [][32]byte) (event.Subscription, error) {

	var commandIdRule []interface{}
	for _, commandIdItem := range commandId {
		commandIdRule = append(commandIdRule, commandIdItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "Executed", commandIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayExecuted)
				if err := _IAxelarGateway.contract.UnpackLog(event, "Executed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseExecuted is a log parse operation binding the contract event 0xa74c8847d513feba22a0f0cb38d53081abf97562cdb293926ba243689e7c41ca.
//
// Solidity: event Executed(bytes32 indexed commandId)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseExecuted(log types.Log) (*IAxelarGatewayExecuted, error) {
	event := new(IAxelarGatewayExecuted)
	if err := _IAxelarGateway.contract.UnpackLog(event, "Executed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayOperatorshipTransferredIterator is returned from FilterOperatorshipTransferred and is used to iterate over the raw logs and unpacked data for OperatorshipTransferred events raised by the IAxelarGateway contract.
type IAxelarGatewayOperatorshipTransferredIterator struct {
	Event *IAxelarGatewayOperatorshipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayOperatorshipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayOperatorshipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayOperatorshipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayOperatorshipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayOperatorshipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayOperatorshipTransferred represents a OperatorshipTransferred event raised by the IAxelarGateway contract.
type IAxelarGatewayOperatorshipTransferred struct {
	NewOperatorsData []byte
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterOperatorshipTransferred is a free log retrieval operation binding the contract event 0x192e759e55f359cd9832b5c0c6e38e4b6df5c5ca33f3bd5c90738e865a521872.
//
// Solidity: event OperatorshipTransferred(bytes newOperatorsData)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterOperatorshipTransferred(opts *bind.FilterOpts) (*IAxelarGatewayOperatorshipTransferredIterator, error) {

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "OperatorshipTransferred")
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayOperatorshipTransferredIterator{contract: _IAxelarGateway.contract, event: "OperatorshipTransferred", logs: logs, sub: sub}, nil
}

// WatchOperatorshipTransferred is a free log subscription operation binding the contract event 0x192e759e55f359cd9832b5c0c6e38e4b6df5c5ca33f3bd5c90738e865a521872.
//
// Solidity: event OperatorshipTransferred(bytes newOperatorsData)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchOperatorshipTransferred(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayOperatorshipTransferred) (event.Subscription, error) {

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "OperatorshipTransferred")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayOperatorshipTransferred)
				if err := _IAxelarGateway.contract.UnpackLog(event, "OperatorshipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOperatorshipTransferred is a log parse operation binding the contract event 0x192e759e55f359cd9832b5c0c6e38e4b6df5c5ca33f3bd5c90738e865a521872.
//
// Solidity: event OperatorshipTransferred(bytes newOperatorsData)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseOperatorshipTransferred(log types.Log) (*IAxelarGatewayOperatorshipTransferred, error) {
	event := new(IAxelarGatewayOperatorshipTransferred)
	if err := _IAxelarGateway.contract.UnpackLog(event, "OperatorshipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayTokenDeployedIterator is returned from FilterTokenDeployed and is used to iterate over the raw logs and unpacked data for TokenDeployed events raised by the IAxelarGateway contract.
type IAxelarGatewayTokenDeployedIterator struct {
	Event *IAxelarGatewayTokenDeployed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayTokenDeployedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayTokenDeployed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayTokenDeployed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayTokenDeployedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayTokenDeployedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayTokenDeployed represents a TokenDeployed event raised by the IAxelarGateway contract.
type IAxelarGatewayTokenDeployed struct {
	Symbol         string
	TokenAddresses common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterTokenDeployed is a free log retrieval operation binding the contract event 0xbf90b5a1ec9763e8bf4b9245cef0c28db92bab309fc2c5177f17814f38246938.
//
// Solidity: event TokenDeployed(string symbol, address tokenAddresses)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterTokenDeployed(opts *bind.FilterOpts) (*IAxelarGatewayTokenDeployedIterator, error) {

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "TokenDeployed")
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayTokenDeployedIterator{contract: _IAxelarGateway.contract, event: "TokenDeployed", logs: logs, sub: sub}, nil
}

// WatchTokenDeployed is a free log subscription operation binding the contract event 0xbf90b5a1ec9763e8bf4b9245cef0c28db92bab309fc2c5177f17814f38246938.
//
// Solidity: event TokenDeployed(string symbol, address tokenAddresses)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchTokenDeployed(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayTokenDeployed) (event.Subscription, error) {

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "TokenDeployed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayTokenDeployed)
				if err := _IAxelarGateway.contract.UnpackLog(event, "TokenDeployed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenDeployed is a log parse operation binding the contract event 0xbf90b5a1ec9763e8bf4b9245cef0c28db92bab309fc2c5177f17814f38246938.
//
// Solidity: event TokenDeployed(string symbol, address tokenAddresses)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseTokenDeployed(log types.Log) (*IAxelarGatewayTokenDeployed, error) {
	event := new(IAxelarGatewayTokenDeployed)
	if err := _IAxelarGateway.contract.UnpackLog(event, "TokenDeployed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayTokenMintLimitUpdatedIterator is returned from FilterTokenMintLimitUpdated and is used to iterate over the raw logs and unpacked data for TokenMintLimitUpdated events raised by the IAxelarGateway contract.
type IAxelarGatewayTokenMintLimitUpdatedIterator struct {
	Event *IAxelarGatewayTokenMintLimitUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayTokenMintLimitUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayTokenMintLimitUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayTokenMintLimitUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayTokenMintLimitUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayTokenMintLimitUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayTokenMintLimitUpdated represents a TokenMintLimitUpdated event raised by the IAxelarGateway contract.
type IAxelarGatewayTokenMintLimitUpdated struct {
	Symbol string
	Limit  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTokenMintLimitUpdated is a free log retrieval operation binding the contract event 0xd99446c1d76385bb5519ccfb5274abcfd5896dfc22405e40010fde217f018a18.
//
// Solidity: event TokenMintLimitUpdated(string symbol, uint256 limit)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterTokenMintLimitUpdated(opts *bind.FilterOpts) (*IAxelarGatewayTokenMintLimitUpdatedIterator, error) {

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "TokenMintLimitUpdated")
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayTokenMintLimitUpdatedIterator{contract: _IAxelarGateway.contract, event: "TokenMintLimitUpdated", logs: logs, sub: sub}, nil
}

// WatchTokenMintLimitUpdated is a free log subscription operation binding the contract event 0xd99446c1d76385bb5519ccfb5274abcfd5896dfc22405e40010fde217f018a18.
//
// Solidity: event TokenMintLimitUpdated(string symbol, uint256 limit)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchTokenMintLimitUpdated(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayTokenMintLimitUpdated) (event.Subscription, error) {

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "TokenMintLimitUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayTokenMintLimitUpdated)
				if err := _IAxelarGateway.contract.UnpackLog(event, "TokenMintLimitUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenMintLimitUpdated is a log parse operation binding the contract event 0xd99446c1d76385bb5519ccfb5274abcfd5896dfc22405e40010fde217f018a18.
//
// Solidity: event TokenMintLimitUpdated(string symbol, uint256 limit)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseTokenMintLimitUpdated(log types.Log) (*IAxelarGatewayTokenMintLimitUpdated, error) {
	event := new(IAxelarGatewayTokenMintLimitUpdated)
	if err := _IAxelarGateway.contract.UnpackLog(event, "TokenMintLimitUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayTokenSentIterator is returned from FilterTokenSent and is used to iterate over the raw logs and unpacked data for TokenSent events raised by the IAxelarGateway contract.
type IAxelarGatewayTokenSentIterator struct {
	Event *IAxelarGatewayTokenSent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayTokenSentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayTokenSent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayTokenSent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayTokenSentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayTokenSentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayTokenSent represents a TokenSent event raised by the IAxelarGateway contract.
type IAxelarGatewayTokenSent struct {
	Sender             common.Address
	DestinationChain   string
	DestinationAddress string
	Symbol             string
	Amount             *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterTokenSent is a free log retrieval operation binding the contract event 0x651d93f66c4329630e8d0f62488eff599e3be484da587335e8dc0fcf46062726.
//
// Solidity: event TokenSent(address indexed sender, string destinationChain, string destinationAddress, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterTokenSent(opts *bind.FilterOpts, sender []common.Address) (*IAxelarGatewayTokenSentIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "TokenSent", senderRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayTokenSentIterator{contract: _IAxelarGateway.contract, event: "TokenSent", logs: logs, sub: sub}, nil
}

// WatchTokenSent is a free log subscription operation binding the contract event 0x651d93f66c4329630e8d0f62488eff599e3be484da587335e8dc0fcf46062726.
//
// Solidity: event TokenSent(address indexed sender, string destinationChain, string destinationAddress, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchTokenSent(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayTokenSent, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "TokenSent", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayTokenSent)
				if err := _IAxelarGateway.contract.UnpackLog(event, "TokenSent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenSent is a log parse operation binding the contract event 0x651d93f66c4329630e8d0f62488eff599e3be484da587335e8dc0fcf46062726.
//
// Solidity: event TokenSent(address indexed sender, string destinationChain, string destinationAddress, string symbol, uint256 amount)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseTokenSent(log types.Log) (*IAxelarGatewayTokenSent, error) {
	event := new(IAxelarGatewayTokenSent)
	if err := _IAxelarGateway.contract.UnpackLog(event, "TokenSent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAxelarGatewayUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the IAxelarGateway contract.
type IAxelarGatewayUpgradedIterator struct {
	Event *IAxelarGatewayUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IAxelarGatewayUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IAxelarGatewayUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IAxelarGatewayUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IAxelarGatewayUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IAxelarGatewayUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IAxelarGatewayUpgraded represents a Upgraded event raised by the IAxelarGateway contract.
type IAxelarGatewayUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_IAxelarGateway *IAxelarGatewayFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*IAxelarGatewayUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _IAxelarGateway.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &IAxelarGatewayUpgradedIterator{contract: _IAxelarGateway.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_IAxelarGateway *IAxelarGatewayFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *IAxelarGatewayUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _IAxelarGateway.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IAxelarGatewayUpgraded)
				if err := _IAxelarGateway.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_IAxelarGateway *IAxelarGatewayFilterer) ParseUpgraded(log types.Log) (*IAxelarGatewayUpgraded, error) {
	event := new(IAxelarGatewayUpgraded)
	if err := _IAxelarGateway.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
