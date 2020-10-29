package core

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var GodAddress = common.HexToAddress("0x444400000000000000000000000000000000000")
var ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

type ovmTransaction struct {
	Timestamp     *big.Int       "json:\"timestamp\""
	BlockNumber   *big.Int       "json:\"blockNumber\""
	L1QueueOrigin uint8          "json:\"l1QueueOrigin\""
	L1TxOrigin    common.Address "json:\"l1TxOrigin\""
	Entrypoint    common.Address "json:\"entrypoint\""
	GasLimit      *big.Int       "json:\"gasLimit\""
	Data          []uint8        "json:\"data\""
}

func toExecutionManagerRun(evm *vm.EVM, msg Message) (Message, error) {
	tx := ovmTransaction{
		evm.Context.Time,
		evm.Context.BlockNumber, // TODO (what's the correct block number?)
		uint8(msg.QueueOrigin().Uint64()),
		*msg.L1MessageSender(),
		*msg.To(),
		big.NewInt(int64(msg.Gas())),
		msg.Data(),
	}

	var abi = vm.OvmExecutionManager.ABI
	var args = []interface{}{
		tx,
		vm.OvmStateManager.Address,
	}

	ret, err := abi.Pack("run", args...)
	if err != nil {
		return nil, err
	}

	outputmsg, err := modMessage(
		msg,
		msg.From(),
		&vm.OvmExecutionManager.Address,
		ret,
	)
	if err != nil {
		return nil, err
	}

	return outputmsg, nil
}

func asOvmMessage(tx *types.Transaction, signer types.Signer) (Message, error) {
	msg, err := tx.AsMessage(signer)
	if err != nil {
		return msg, err
	}

	if msg.From() == GodAddress {
		return msg, nil
	}

	v, r, s := tx.RawSignatureValues()
	v = new(big.Int).Mod(v, big.NewInt(256))
	var data = new(bytes.Buffer)

	var sigtype = getSignatureType(msg)

	var target common.Address
	if tx.To() == nil {
		target = ZeroAddress
	} else {
		target = *tx.To()
	}

	// Signature type
	data.WriteByte(byte(sigtype)) // 1 byte: 00 == EOACreate, 01 == EIP 155, 02 == ETH Sign Message

	// Signature data
	data.Write(v.FillBytes(make([]byte, 1, 1)))   // 1 byte: Signature `v` parameter
	data.Write(r.FillBytes(make([]byte, 32, 32))) // 32 bytes: Signature `r` parameter
	data.Write(s.FillBytes(make([]byte, 32, 32))) // 32 bytes: Signature `s` parameter

	if sigtype == 0 {
		// EOACreate: Encode the transaction hash.
		data.Write(signer.Hash(tx).Bytes()) // 32 bytes: Transaction hash
	} else {
		// EIP 155 or ETH Sign Message: Encode the full transaction data.
		data.Write(big.NewInt(int64(msg.Nonce())).FillBytes(make([]byte, 2, 2))) // 2 bytes: Nonce
		data.Write(big.NewInt(int64(msg.Gas())).FillBytes(make([]byte, 3, 3)))   // 3 bytes: Gas limit
		data.Write(msg.GasPrice().FillBytes(make([]byte, 1, 1)))                 // 1 byte: Gas price
		data.Write(tx.ChainId().FillBytes(make([]byte, 4, 4)))                   // 4 bytes: Chain ID
		data.Write(target.Bytes())                                               // 20 bytes: Target address
		data.Write(msg.Data())                                                   // ?? bytes: Transaction data
	}

	decompressor := vm.OvmStateDump.Accounts["OVM_SequencerMessageDecompressor"]

	outmsg, err := modMessage(
		msg,
		GodAddress,
		&(decompressor.Address),
		data.Bytes(),
	)

	if err != nil {
		return msg, err
	}

	return outmsg, nil
}

func EncodeFakeMessage(
	msg Message,
) (Message, error) {
	var input = []interface{}{
		big.NewInt(int64(msg.Gas())),
		msg.To(),
		msg.Data(),
	}

	var abi = vm.OvmStateDump.Accounts["mockOVM_ECDSAContractAccount"].ABI
	output, err := abi.Pack("kall", input...)
	if err != nil {
		return nil, err
	}

	var from = msg.From()
	return modMessage(
		msg,
		from,
		&from,
		output,
	)
}

func modMessage(
	msg Message,
	from common.Address,
	to *common.Address,
	data []byte,
) (Message, error) {
	queueOrigin, err := getQueueOrigin(msg.QueueOrigin())
	if err != nil {
		return nil, err
	}

	outmsg := types.NewMessage(
		from,
		to,
		msg.Nonce(),
		msg.Value(),
		msg.Gas(),
		msg.GasPrice(),
		data,
		false,
		msg.L1MessageSender(),
		msg.L1RollupTxId(),
		queueOrigin,
		msg.SignatureHashType(),
	)

	return outmsg, nil
}

func getSignatureType(
	msg Message,
) uint8 {
	if msg.SignatureHashType() == 0 {
		return 1
	} else if msg.SignatureHashType() == 1 {
		return 2
	} else {
		return 0
	}
}

func getQueueOrigin(
	queueOrigin *big.Int,
) (types.QueueOrigin, error) {
	if queueOrigin.Cmp(big.NewInt(0)) == 0 {
		return types.QueueOriginSequencer, nil
	} else if queueOrigin.Cmp(big.NewInt(1)) == 0 {
		return types.QueueOriginL1ToL2, nil
	} else {
		return types.QueueOriginSequencer, fmt.Errorf("invalid queue origin: %d", queueOrigin)
	}
}