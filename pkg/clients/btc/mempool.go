package btc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	maxRetries = 20
	retryDelay = 5 * time.Second
)

type BitcoinTransaction struct {
	TxID string `json:"txid"`
	Vout []struct {
		Value float64 `json:"value"`
	} `json:"vout"`
	Status struct {
		BlockHeight int64 `json:"block_height"`
	} `json:"status"`
}

func GetMempoolTx(txID string, network string) (*btcjson.GetTransactionResult, error) {
	prefix := ""
	if network == "testnet" {
		prefix = "/testnet"
	}
	endpoint := fmt.Sprintf("%s%s/api/tx/%s", viper.GetString("MEMPOOL_API"), prefix, txID)

	for i := 0; i <= maxRetries; i++ {
		resp, err := http.Get(endpoint)
		if err != nil {
			fmt.Printf("Attempt %d failed: %v\n", i+1, err)
			if i < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("all retries failed: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		var tx BitcoinTransaction
		if err := json.Unmarshal(body, &tx); err != nil {
			fmt.Printf("Failed to parse JSON: %v\n", err)
			return nil, err
		}

		if tx.TxID != "" {
			return &btcjson.GetTransactionResult{
				Amount:     tx.Vout[0].Value,
				TxID:       tx.TxID,
				BlockIndex: tx.Status.BlockHeight,
			}, nil
		}
	}

	return nil, fmt.Errorf("transaction not found after %d attempts", maxRetries)
}

func (c *BtcClient) GetAddressTxsUtxo(taprootAddress string) ([]wire.TxIn, error) {
	url := fmt.Sprintf("%s/address/%s/utxo", c.btcConfig.MempoolUrl, taprootAddress)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	//log.Debug().Msgf("[BtcClient] [GetAddressTxsUtxo] body: %v", string(body))

	var utxos []Utxo
	if err := json.Unmarshal(body, &utxos); err != nil {
		return nil, fmt.Errorf("failed to decode UTXOs: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [GetAddressTxsUtxo] utxos: %v", utxos)

	txIns := make([]wire.TxIn, 0, len(utxos))
	for _, utxo := range utxos {
		hash, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %s: %w", utxo.Txid, err)
		}

		txIn := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *hash,
				Index: utxo.Vout,
			},
		}
		txIns = append(txIns, txIn)
	}

	return txIns, nil
}
