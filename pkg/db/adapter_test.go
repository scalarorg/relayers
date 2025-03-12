package db_test

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"

// 	"github.com/rs/zerolog/log"
// 	"github.com/scalarorg/data-models/chains"
// 	"github.com/scalarorg/data-models/scalarnet"
// 	"github.com/scalarorg/relayers/pkg/db"
// 	"github.com/scalarorg/relayers/pkg/events"
// 	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
// 	"github.com/stretchr/testify/require"
// 	"github.com/testcontainers/testcontainers-go"
// 	"github.com/testcontainers/testcontainers-go/modules/postgres"
// 	"github.com/testcontainers/testcontainers-go/wait"
// 	postgresDriver "gorm.io/driver/postgres"
// 	"gorm.io/gorm"
// )

// const (
// 	CONTRACT_CALL_WITH_TOKEN_EVENT_ID = "0x4a21de48f14ae787a11cce77f6232fe52308590791829b54ecc2f82f36a2468f-42"
// 	TOKEN_SENT_EVENT_ID               = "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291-735"
// 	BATCH_COMMAND_ID                  = "b4bfce77284eeb906bac44ced14b5c0f0c87fd2c0542a1f681c2424e9d129cea"
// 	COMMAND_ID                        = "0x4a21de48f14ae787a11cce77f6232fe52308590791829b54ecc2f82f36a2468f"
// 	TX_HASH_BTC                       = "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291"
// )

// var (
// 	dbAdapter *db.DatabaseAdapter
// 	cleanup   func()
// )

// func TestMain(m *testing.M) {
// 	var err error
// 	dbAdapter, cleanup, err = SetupTestDB(make(chan *events.EventEnvelope, 100), 100)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to setup test db")
// 		os.Exit(1)
// 	}
// 	os.Exit(m.Run())
// }

// func SetupTestDB(busEventChan chan *events.EventEnvelope, receiverChanBufSize int) (*db.DatabaseAdapter, func(), error) {
// 	ctx := context.Background()

// 	dbName := "test_db"
// 	dbUser := "test_user"
// 	dbPassword := "test_password"

// 	postgresContainer, err := postgres.Run(ctx,
// 		"postgres:16-alpine",
// 		postgres.WithDatabase(dbName),
// 		postgres.WithUsername(dbUser),
// 		postgres.WithPassword(dbPassword),
// 		testcontainers.WithWaitStrategy(
// 			wait.ForLog("database system is ready to accept connections").
// 				WithOccurrence(2).
// 				WithStartupTimeout(5*time.Second)),
// 	)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	// Get the container's host and port
// 	host, err := postgresContainer.Host(ctx)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	port, err := postgresContainer.MappedPort(ctx, "5432")
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
// 		host, dbUser, dbPassword, dbName, port.Int())

// 	postgresDb, err := gorm.Open(postgresDriver.Open(dsn), &gorm.Config{})
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	// Auto migrate the schema
// 	err = db.RunMigrations(postgresDb)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	createMockData(postgresDb)

// 	// Create test DatabaseAdapter
// 	testAdapter := &db.DatabaseAdapter{
// 		PostgresClient: postgresDb,
// 		// BusEventChan:         busEventChan,
// 		// BusEventReceiverChan: make(chan *types.EventEnvelope, receiverChanBufSize),
// 	}

// 	// Return cleanup function
// 	cleanup := func() {
// 		postgresContainer.Terminate(ctx)
// 	}

// 	return testAdapter, cleanup, nil
// }

// func createMockData(db *gorm.DB) {
// 	tokenSent := chains.TokenSent{
// 		EventID:              "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291-735",
// 		TxHash:               "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		BlockNumber:          66084,
// 		LogIndex:             735,
// 		SourceChain:          "bitcoin|4",
// 		SourceAddress:        "tb1q2rwweg2c48y8966qt4fzj0f4zyg9wty7tykzwg",
// 		DestinationChain:     "evm|11155111",
// 		DestinationAddress:   "982321eb5693cdbaadffe97056bece07d09ba49f",
// 		TokenContractAddress: "563fea1c8c36f3f97a963de9e1d05f78f84c64ca",
// 		Amount:               1000,
// 		Symbol:               "tBtc",
// 		Status:               chains.TokenSentStatusPending,
// 	}
// 	tokenSentApproved := scalarnet.TokenSentApproved{
// 		EventID:            "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291-735",
// 		SourceChain:        "bitcoin|4",
// 		SourceAddress:      "tb1q2rwweg2c48y8966qt4fzj0f4zyg9wty7tykzwg",
// 		DestinationChain:   "evm|11155111",
// 		DestinationAddress: "982321eb5693cdbaadffe97056bece07d09ba49f",
// 		SourceTxHash:       "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		BlockNumber:        66084,
// 		LogIndex:           735,
// 		Amount:             1000,
// 		Symbol:             "tBtc",
// 		Status:             "approved",
// 		CommandID:          "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		TransferID:         1,
// 	}
// 	contractCallWithToken := chains.ContractCallWithToken{
// 		ContractCall: chains.ContractCall{
// 			EventID:          CONTRACT_CALL_WITH_TOKEN_EVENT_ID,
// 			TxHash:           "0x4a21de48f14ae787a11cce77f6232fe52308590791829b54ecc2f82f36a2468f",
// 			BlockNumber:      7538226,
// 			LogIndex:         42,
// 			SourceChain:      "evm|11155111",
// 			SourceAddress:    "0x8b73c6c3f60ac6f45bb6a7d2a0080af829c76e43",
// 			PayloadHash:      "5c28ec958ca65352fb9d46ce7248e0a57de240677663d84e268481717904b563",
// 			DestinationChain: "bitcoin|4",
// 			Status:           chains.ContractCallStatusPending,
// 		},
// 		Symbol: "tBtc",
// 		Amount: 1000,
// 	}
// 	contractCallApprovedWithMint := scalarnet.ContractCallApprovedWithMint{
// 		ContractCallApproved: scalarnet.ContractCallApproved{
// 			EventID:          CONTRACT_CALL_WITH_TOKEN_EVENT_ID,
// 			TxHash:           "0x4a21de48f14ae787a11cce77f6232fe52308590791829b54ecc2f82f36a2468f",
// 			SourceChain:      "evm|11155111",
// 			DestinationChain: "bitcoin|4",
// 			CommandID:        COMMAND_ID,
// 			Sender:           "0x982321eb5693cdbAadFfe97056BEce07D09Ba49f",
// 			PayloadHash:      "5c28ec958ca65352fb9d46ce7248e0a57de240677663d84e268481717904b563",
// 		},
// 		Symbol: "tBtc",
// 		Amount: 1000,
// 	}
// 	command := scalarnet.Command{
// 		CommandID:      COMMAND_ID,
// 		BatchCommandID: BATCH_COMMAND_ID,
// 		ChainID:        "bitcoin|4",
// 		Status:         scalarnet.CommandStatusPending,
// 	}
// 	db.Save(&tokenSent)
// 	db.Save(&tokenSentApproved)
// 	db.Save(&contractCallWithToken)
// 	db.Save(&contractCallApprovedWithMint)
// 	db.Save(&command)
// }

// func TestSaveMintTokenCommandExecuted(t *testing.T) {
// 	cmdExecuted := chains.CommandExecuted{
// 		SourceChain: "bitcoin|4",
// 		CommandID:   "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		Address:     "tb1q2rwweg2c48y8966qt4fzj0f4zyg9wty7tykzwg",
// 		TxHash:      "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		BlockNumber: 66084,
// 		LogIndex:    735,
// 	}
// 	err := dbAdapter.SaveCommandExecuted(&cmdExecuted, &chainstypes.CommandResponse{Type: "mintToken"}, "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291")
// 	tokenSent := chains.TokenSent{}
// 	dbAdapter.PostgresClient.Find(&chains.TokenSent{}).Where("event_id = ?", TOKEN_SENT_EVENT_ID).First(&tokenSent)
// 	require.NoError(t, err)
// 	require.Equal(t, chains.TokenSentStatusSuccess, tokenSent.Status)
// 	cleanup()
// }

// func TestSaveContractCallApproved(t *testing.T) {
// 	cmdExecuted := chains.CommandExecuted{
// 		SourceChain: "bitcoin|4",
// 		CommandID:   "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		Address:     "tb1q2rwweg2c48y8966qt4fzj0f4zyg9wty7tykzwg",
// 		TxHash:      "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291",
// 		BlockNumber: 66084,
// 		LogIndex:    735,
// 	}
// 	err := dbAdapter.SaveCommandExecuted(&cmdExecuted, &chainstypes.CommandResponse{Type: "approveContractCallWithMint"}, "2919df3249096c9b166ce5f16e7dc55e94a141b50f0941270f2f52187640c291")
// 	require.NoError(t, err)
// 	cleanup()
// }

// func TestUpdateBroadcastedCommands(t *testing.T) {
// 	err := dbAdapter.UpdateBroadcastedCommands("bitcoin|4",
// 		BATCH_COMMAND_ID,
// 		[]string{COMMAND_ID},
// 		TX_HASH_BTC)
// 	require.NoError(t, err)
// 	contractCallWithToken := chains.ContractCallWithToken{}
// 	dbAdapter.PostgresClient.Find(&chains.ContractCallWithToken{}).
// 		Where("event_id = ?", CONTRACT_CALL_WITH_TOKEN_EVENT_ID).
// 		First(&contractCallWithToken)
// 	require.Equal(t, chains.ContractCallStatusExecuting, contractCallWithToken.Status)
// 	cleanup()
// }

// func TestUpdateBtcExecutedCommands(t *testing.T) {
// 	err := dbAdapter.UpdateBroadcastedCommands("bitcoin|4",
// 		BATCH_COMMAND_ID,
// 		[]string{COMMAND_ID},
// 		TX_HASH_BTC)
// 	require.NoError(t, err)
// 	err = dbAdapter.UpdateBtcExecutedCommands("bitcoin|4", []string{TX_HASH_BTC})
// 	require.NoError(t, err)
// 	contractCallWithToken := chains.ContractCallWithToken{}
// 	dbAdapter.PostgresClient.Find(&chains.ContractCallWithToken{}).
// 		Where("event_id = ?", CONTRACT_CALL_WITH_TOKEN_EVENT_ID).
// 		First(&contractCallWithToken)
// 	require.Equal(t, chains.ContractCallStatusSuccess, contractCallWithToken.Status)
// 	cleanup()
// }
