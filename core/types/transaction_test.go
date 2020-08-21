// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// The values in those tests are from the Transaction Tests
// at github.com/ethereum/tests.
var (
	sender               = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	l1RollupTxId         = hexutil.Uint64(1)
	emptyTx              = NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(0), 0, big.NewInt(0), nil, &sender, nil, nil)
	emptyTxEmptyL1Sender = NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(0), 0, big.NewInt(0), nil, nil, nil, nil)

	rightvrsTx, _ = NewTransaction(3, common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"), big.NewInt(10), 2000, big.NewInt(1), common.FromHex("5544"), nil, nil, nil).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	rightvrsTxWithL1Sender, _ = NewTransaction(3, common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"), big.NewInt(10), 2000, big.NewInt(1), common.FromHex("5544"), &sender, nil, &SighashEIP155).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	rightvrsTxWithL1RollupTxId, _ = NewTransaction(3, common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"), big.NewInt(10), 2000, big.NewInt(1), common.FromHex("5544"), nil, &l1RollupTxId, &SighashEIP155).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	emptyTxSighashEthSign = NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(0), 0, big.NewInt(0), nil, &sender, nil, &SighashEthSign)
)

func TestTransactionSigHash(t *testing.T) {
	var homestead HomesteadSigner
	if homestead.Hash(emptyTx) != common.HexToHash("c775b99e7ad12f50d819fcd602390467e28141316969f4b57f0626f74fe3b386") {
		t.Errorf("empty transaction hash mismatch, got %x", emptyTx.Hash())
	}
	if homestead.Hash(rightvrsTx) != common.HexToHash("fe7a79529ed5f7c3375d06b26b186a8644e0e16c373d7a12be41c62d6042b77a") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", rightvrsTx.Hash())
	}
}

func TestTransactionEncode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(rightvrsTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	should := common.FromHex("f86103018207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a8255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(txb, should) {
		t.Errorf("encoded RLP mismatch, got %x", txb)
	}

	txc, err := rlp.EncodeToBytes(rightvrsTxWithL1Sender)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if bytes.Equal(txc, should) {
		t.Errorf("RLP encoding with L1MessageSender should be different than without. Got %x", txc)
	}

	txd, err := rlp.EncodeToBytes(rightvrsTxWithL1RollupTxId)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if bytes.Equal(txd, should) {
		t.Errorf("RLP encoding with L1MessageSender should be different than without. Got %x", txd)
	}

	// RLP encode both the empty transaction and the empty transaction that
	// uses the `eth_sign` signature hash and assert that they are not the same.
	// The signature hash flag must be included in the RLP encoding only when it
	// is defined so that it can be persisted in the database. When the
	// SignatureHashType is `nil`, it is not included in the RLP serialization.
	txe, err := rlp.EncodeToBytes(emptyTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	txf, err := rlp.EncodeToBytes(emptyTxSighashEthSign)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if bytes.Equal(txe, txf) {
		t.Error("RLP encoding with SighashEthSign should be different than without")
	}
}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	t, err := &tx, rlp.Decode(bytes.NewReader(data), &tx)

	return t, err
}

func defaultTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func TestRecipientEmpty(t *testing.T) {
	_, addr := defaultTestKey()
	tx, err := decodeTx(common.Hex2Bytes("f8498080808080011ca09b16de9d5bdee2cf56c28d16275a4da68cd30273e2525f3959f5d62557489921a0372ebd8fb3345f7db7b5a86d42e24d36e983e259b0664ceb8c227ec9af572f3d"))
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}
}

func TestRecipientNormal(t *testing.T) {
	_, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f85d80808094000000000000000000000000000000000000000080011ca0527c0d8f5c63f7b9f41324a7c8a563ee1190bcbf0dac8ab446291bdbf32f5c79a0552c4ef0a09a04395074dab9ed34d3fbfb843c2f2546cc30fe89ec143ca94ca6"))
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}
}

// Tests that transactions can be correctly sorted according to their price in
// decreasing order, but at the same time with increasing nonces when issued by
// the same account.
func TestTransactionPriceNonceSort(t *testing.T) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 25)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}

	signer := HomesteadSigner{}
	// Generate a batch of transactions with overlapping values, but shifted nonces
	groups := map[common.Address]Transactions{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for i := 0; i < 25; i++ {
			tx, _ := SignTx(NewTransaction(uint64(start+i), common.Address{}, big.NewInt(100), 100, big.NewInt(int64(start+i)), nil, nil, nil, &SighashEIP155), signer, key)
			groups[addr] = append(groups[addr], tx)
		}
	}
	// Sort the transactions and cross check the nonce ordering
	txset := NewTransactionsByPriceAndNonce(signer, groups)

	txs := Transactions{}
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		txs = append(txs, tx)
		txset.Shift()
	}
	if len(txs) != 25*25 {
		t.Errorf("expected %d transactions, found %d", 25*25, len(txs))
	}
	for i, txi := range txs {
		fromi, _ := Sender(signer, txi)

		// Make sure the nonce order is valid
		for j, txj := range txs[i+1:] {
			fromj, _ := Sender(signer, txj)

			if fromi == fromj && txi.Nonce() > txj.Nonce() {
				t.Errorf("invalid nonce ordering: tx #%d (A=%x N=%v) < tx #%d (A=%x N=%v)", i, fromi[:4], txi.Nonce(), i+j, fromj[:4], txj.Nonce())
			}
		}

		// If the next tx has different from account, the price must be lower than the current one
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := Sender(signer, next)
			if fromi != fromNext && txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
		}
	}
}

// TestTransactionJSON tests serializing/de-serializing to/from JSON.
func TestTransactionJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewOVMSigner(common.Big1)

	transactions := make([]*Transaction, 0, 50)
	for i := uint64(0); i < 25; i++ {
		var tx *Transaction
		switch i % 2 {
		case 0:
			tx = NewTransaction(i, common.Address{1}, common.Big0, 1, common.Big2, []byte("abcdef"), &sender, &l1RollupTxId, &SighashEIP155)
		case 1:
			tx = NewContractCreation(i, common.Big0, 1, common.Big2, []byte("abcdef"), nil, nil)
		}
		transactions = append(transactions, tx)

		signedTx, err := SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}

		transactions = append(transactions, signedTx)
	}

	for _, tx := range transactions {
		data, err := json.Marshal(tx)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var parsedTx *Transaction
		if err := json.Unmarshal(data, &parsedTx); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
		if tx.Hash() != parsedTx.Hash() {
			t.Errorf("parsed tx differs from original tx, want %v, got %v", tx, parsedTx)
		}
		if tx.ChainId().Cmp(parsedTx.ChainId()) != 0 {
			t.Errorf("invalid chain id, want %d, got %d", tx.ChainId(), parsedTx.ChainId())
		}
		if tx.L1MessageSender() == nil && parsedTx.L1MessageSender() != nil || tx.L1MessageSender() != nil && parsedTx.L1MessageSender() == nil || (tx.L1MessageSender() != nil && parsedTx.L1MessageSender() != nil && *tx.L1MessageSender() != *parsedTx.L1MessageSender()) {
			t.Errorf("invalid L1MessageSender, want %x, got %x", tx.L1MessageSender(), parsedTx.L1MessageSender())
		}
		if tx.L1RollupTxId() == nil && parsedTx.L1RollupTxId() != nil || tx.L1RollupTxId() != nil && parsedTx.L1RollupTxId() == nil || (tx.L1RollupTxId() != nil && parsedTx.L1RollupTxId() != nil && *tx.L1RollupTxId() != *parsedTx.L1RollupTxId()) {
			t.Errorf("invalid L1RollupTxId, want %x, got %x", tx.L1RollupTxId(), parsedTx.L1RollupTxId())
		}
	}
}

// Tests that OVM metadata has no impact on hash
func TestOVMMetaDataHash(t *testing.T) {
	if rightvrsTx.Hash() != rightvrsTxWithL1Sender.Hash() {
		t.Errorf("L1MessageSender, should not affect the hash, want %x, got %x with L1MessageSender", rightvrsTx.Hash(), rightvrsTxWithL1Sender.Hash())
	}

	if rightvrsTx.Hash() != rightvrsTxWithL1RollupTxId.Hash() {
		t.Errorf("L1RollupTxId, should not affect the hash, want %x, got %x with L1RollupTxId", rightvrsTx.Hash(), rightvrsTxWithL1RollupTxId.Hash())
	}

	if emptyTx.Hash() != emptyTxEmptyL1Sender.Hash() {
		t.Errorf("L1MessageSender, should not affect the hash, want %x, got %x with L1MessageSender", emptyTx.Hash(), emptyTxEmptyL1Sender.Hash())
	}

	if emptyTx.Hash() != emptyTxSighashEthSign.Hash() {
		t.Errorf("SignatureHashType, should not affect the hash, want %x, got %x with SighashEthSign", emptyTx.Hash(), emptyTxSighashEthSign.Hash())
	}
}
