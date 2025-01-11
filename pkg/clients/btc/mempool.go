package btc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/btcsuite/btcd/btcjson"
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

func (c *BtcClient) GetAddressTxsUtxo(taprootAddress string, totalAmount uint64) ([]Utxo, error) {
	if taprootAddress == "" {
		return nil, fmt.Errorf("taproot address cannot be empty")
	}
	if totalAmount == 0 {
		return nil, fmt.Errorf("total amount must be greater than 0")
	}

	utxos, err := c.getListOfUTXOs(taprootAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}

	utxos = sortUTXOsByValue(utxos)

	selectedUTXOs := make([]Utxo, 0, len(utxos))
	totalValue := uint64(0)

	for _, utxo := range utxos {
		selectedUTXOs = append(selectedUTXOs, utxo)
		totalValue += utxo.Value
		if totalValue >= totalAmount {
			break
		}
	}

	if totalValue < totalAmount {
		return nil, fmt.Errorf("insufficient funds: available balance %d is less than required amount %d",
			totalValue, totalAmount)
	}

	return selectedUTXOs, nil
}

func (c BtcClient) getListOfUTXOs(taprootAddress string) ([]Utxo, error) {
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
	return utxos, nil

}

func sortUTXOsByValue(utxos []Utxo) []Utxo {
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Value > utxos[j].Value
	})
	return utxos
}
