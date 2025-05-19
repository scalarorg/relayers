package scalar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/scalar-core/utils"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"
)

// https://github.com/cosmos/cosmos-sdk/blob/main/client/tx/tx.go#L31

type Broadcaster struct {
	network         *cosmos.NetworkClient
	pendingCommands *PendingCommands //Keep reference to the pending commands for store sign command request tx hashå
	messageBuffer   *MessageBuffer
	mutex           sync.Mutex
	isRunning       bool
	period          time.Duration
	batchSize       int //Number messages to broadcast in a transaction
	cycleCount      int //Log message is printed every 1000 messages
}

func NewBroadcaster(network *cosmos.NetworkClient, pendingCommands *PendingCommands, broadcastPeriod time.Duration, batchSize int) *Broadcaster {
	return &Broadcaster{
		network:         network,
		pendingCommands: pendingCommands,
		messageBuffer:   NewMessageBuffer(),
		period:          broadcastPeriod,
		batchSize:       batchSize,
		cycleCount:      0,
	}
}

// func createQueue(network *cosmos.NetworkClient, broadcastPeriod time.Duration, batchSize int) (*queue.Queue, error) {
// 	worker := NewWorker(
// 		network,
// 		broadcastPeriod,
// 		batchSize,
// 	)
// 	queue, err := queue.NewQueue(
// 		queue.WithLogger(zerolog.New()),
// 		queue.WithWorkerCount(1),
// 		queue.WithWorker(worker),
// 	)
// 	return queue, err
// }

func (b *Broadcaster) Start(ctx context.Context) error {
	b.mutex.Lock()
	if b.isRunning {
		b.mutex.Unlock()
		return nil
	}
	b.isRunning = true
	b.mutex.Unlock()
	go b.broadcastLoop(ctx)
	//b.queue.Start()
	return nil
}

// Stop gracefully stops the broadcaster
func (b *Broadcaster) Stop() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.isRunning = false
	//b.queue.Release()
}

// QueueMsg adds a message to the broadcasting queue
func (b *Broadcaster) QueueTxMsg(msg types.Msg) error {
	if !b.isRunning {
		return fmt.Errorf("broadcaster is not running")
	}
	log.Debug().Msgf("[Broadcaster] [QueueTxMsg] enqueue message %T", msg)
	b.messageBuffer.AddNewMsg(msg)
	return nil
}
func (b *Broadcaster) QueueSignCommandReq(chain string, msg types.Msg) error {
	if !b.isRunning {
		return fmt.Errorf("broadcaster is not running")
	}
	return b.messageBuffer.AddSignCommandReq(chain, msg)
}

func (c *Broadcaster) ConfirmEvmTxs(chainName string, txIds []string) error {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[Broadcaster] [ConfirmEvmTxs] Enqueue for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]chainsExported.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainsExported.Hash(common.HexToHash(txId))
	}
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
	return c.QueueTxMsg(msg)
}

func (c *Broadcaster) ConfirmBtcTxs(chainName string, txIds []string) error {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[Broadcaster] [ConfirmBtcTxs] Enqueue for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]chainsExported.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainsExported.Hash(common.HexToHash(txId))
	}
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
	return c.QueueTxMsg(msg)
}

func (c *Broadcaster) ConfirmRedeemTxRequest(redeemRequest events.ConfirmRedeemTxRequest) error {
	txHashs := make([]chainsExported.Hash, len(redeemRequest.TxHashs))
	for i, txId := range redeemRequest.TxHashs {
		txHashs[i] = chainsExported.Hash(common.HexToHash(txId))
	}
	msg := covtypes.NewConfirmRedeemTxRequest(
		c.network.GetAddress(),
		redeemRequest.Chain,
		redeemRequest.GroupUid,
		txHashs,
	)
	return c.QueueTxMsg(msg)
}

func (c *Broadcaster) InitializeUtxoRequest(chain string, height uint64) error {
	log.Debug().Str("Chain", chain).Uint64("Height", height).Msg("[Broadcaster] [InitializeUtxoRequest] Initialize utxo request")
	msg := covtypes.NewInitializeUtxoRequest(
		c.network.GetAddress(),
		nexus.ChainName(chain),
		height,
	)
	return c.QueueTxMsg(msg)
}

//	func (c *Broadcaster) NewBtcBlock(blockHeight events.ChainBlockHeight) error {
//		msg := covtypes.NewUpdateUtxoListsRequest(
//			c.network.GetAddress(),
//			blockHeight.Chain,
//			blockHeight.Height,
//		)
//		return c.QueueTxMsg(msg)
//	}
func (c *Broadcaster) AddSignEvmCommandsRequest(destinationChain string) error {
	log.Debug().Str("Chain", destinationChain).Msg("[Broadcaster] [AddSignEvmCommandsRequest] Add SignEvmCommandsRequest to buffer")
	req := chainstypes.NewSignCommandsRequest(
		c.network.GetAddress(),
		destinationChain)
	return c.QueueSignCommandReq(destinationChain, req)
}

func (c *Broadcaster) ConfirmTokenDeployed(tokenDeployed *chains.TokenDeployed) error {
	log.Debug().Str("TxHash", tokenDeployed.TxHash).
		Str("TokenAddress", tokenDeployed.TokenAddress).Msgf("[Broadcaster] [ConfirmTokenDeployed] Confirm token deployed")
	msg := chainstypes.NewConfirmTokenRequest(c.network.GetAddress(),
		tokenDeployed.Chain,
		chainstypes.Asset{
			Chain:  nexus.ChainName(tokenDeployed.Chain),
			Symbol: tokenDeployed.Symbol,
		},
		common.HexToHash(tokenDeployed.TxHash))
	return c.QueueTxMsg(msg)
}

func (c *Broadcaster) ConfirmSwitchedPhase(switchedPhase *chains.SwitchedPhase) error {
	log.Debug().Str("TxHash", switchedPhase.TxHash).
		Str("Chain", switchedPhase.Chain).
		Str("CustodianGroupUid", switchedPhase.CustodianGroupUid).
		Uint64("SessionSequence", switchedPhase.SessionSequence).
		Uint8("From", switchedPhase.From).
		Uint8("To", switchedPhase.To).Msgf("[Broadcaster] [ConfirmSwitchedPhase] Confirm switched phase")
	msg := covtypes.NewConfirmSwitchedPhaseRequest(
		c.network.GetAddress(),
		switchedPhase.Chain,
		switchedPhase.CustodianGroupUid,
		switchedPhase.TxHash,
	)
	return c.QueueTxMsg(msg)
}

// Add SignBtcCommandsRequest to buffer
// Return true if the request is added to buffer, false if the request is already in buffer
func (c *Broadcaster) AddSignUpcCommandsRequest(destinationChain string) error {
	req := chainstypes.NewSignBtcCommandsRequest(
		c.network.GetAddress(),
		destinationChain)

	err := c.QueueSignCommandReq(destinationChain, req)
	return err
}

func (c *Broadcaster) CreatePendingTransfersRequest(chain string) error {
	req := chainstypes.CreatePendingTransfersRequest{
		Sender: c.network.GetAddress(),
		Chain:  nexus.ChainName(chain),
	}
	return c.QueueTxMsg(&req)
}

// try broadcast fist messages in the buffer
func (b *Broadcaster) broadcastMsgs(ctx context.Context) error {
	txMsgs := b.messageBuffer.RetrieveNewMsgs(b.batchSize)
	failedMsgs := b.messageBuffer.RetrieveFailedMsgs()
	signCommandReqs := b.messageBuffer.RetrieveAllSignCommandReqs()
	if len(txMsgs) == 0 && len(signCommandReqs) == 0 && len(failedMsgs) == 0 {
		if b.cycleCount >= 1000 {
			log.Debug().Msg("[Broadcaster] No messages to broadcast")
			b.cycleCount = 0
		}
		return nil
	} else {
		log.Debug().Int("txMsgs", len(txMsgs)).Int("signCommandReqs", len(signCommandReqs)).Msg("[Broadcaster] [broadcastMsgs] found pending commands in buffer")
	}
	//Broadcast txMsgs
	if len(txMsgs) > 0 {
		log.Debug().Msgf("[Broadcaster] [broadcastMsgs] broadcasting %d messages", len(txMsgs))
		resp, err := b.network.SignAndBroadcastMsgs(ctx, txMsgs...)
		if err != nil {
			log.Error().Err(err).Msgf("[Broadcaster] Failed to broadcast %d messages", len(txMsgs))
			b.messageBuffer.AddFailedMsg(txMsgs...)
		} else if resp.Code == 0 {
			log.Debug().
				Str("tx_hash", resp.TxHash).
				Msgf("[Broadcaster] Successfully broadcasted %d messages in single tx", len(txMsgs))
		}
	}
	//Broadcast failedMsgs
	if len(failedMsgs) > 0 {
		log.Debug().Msgf("[Broadcaster] [broadcastMsgs] retry broadcasting %d failed messages", len(failedMsgs))
		for _, msg := range failedMsgs {
			resp, err := b.network.SignAndBroadcastMsgs(ctx, msg)
			if err != nil {
				log.Error().Err(err).Msgf("[Broadcaster] Failed to retry broadcast msg %T, %++v", msg, msg)
				return err
			}
			if resp.Code == 0 {
				log.Debug().
					Str("tx_hash", resp.TxHash).
					Msgf("[Broadcaster] Successfully retry broadcast msg %T, %++v", msg, msg)
			}
		}
	}
	//Broadcast signCommandReqs
	for chain, msg := range signCommandReqs {
		resp, err := b.network.SignAndBroadcastMsgs(ctx, msg)
		if err != nil {
			log.Warn().Err(err).Msgf("[Broadcaster] Failed to broadcast signCommandReqs %T for chain %s", msg, chain)
			return err
		} else if resp.Code == 0 {
			log.Debug().
				Str("chain", chain).
				Str("tx_hash", resp.TxHash).
				Msgf("[Broadcaster] Successfully broadcasted signCommandReqs %T", msg)
			b.pendingCommands.StoreSignRequest(chain, resp.TxHash)
		}
	}

	// if resp.Code == 0 {
	// 	log.Info().
	// 		Int("msg_count", len(msgs)).
	// 		Str("tx_hash", resp.TxHash).
	// 		Int("remain_buffer_size", len(b.buffers)).
	// 		Msg("[Broadcaster] Successfully broadcasted messages")
	// 	for _, msg := range msgs {
	// 		switch value := msg.(type) {
	// 		case *chainstypes.SignCommandsRequest:
	// 			log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignEvmPendingCommandsRequest")
	// 			b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
	// 			b.CleanPendingCommandRequests(string(value.Chain))
	// 		case *chainstypes.SignPsbtCommandRequest:
	// 			log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignPsbtCommandRequest")
	// 			b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
	// 			b.CleanPendingCommandRequests(string(value.Chain))
	// 		case *chainstypes.SignBtcCommandsRequest:
	// 			log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignUpcCommandRequest")
	// 			b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
	// 			b.CleanPendingCommandRequests(string(value.Chain))
	// 		default:
	// 			//log.Debug().Msgf("[Broadcaster] [successfully broadcasted]: %v of type %T", msg, msg)
	// 		}
	// 	}
	// 	msgs = nil
	// } else {
	// 	log.Error().
	// 		Uint32("code", resp.Code).
	// 		Str("raw_log", resp.RawLog).
	// 		Msg("[Broadcaster] Broadcast messages failed put back to the buffer")
	// 	// put messages back to the buffer
	// 	b.pushFailedMsgBackToBuffer(msgs)
	// }
	return nil
}

// broadcastLoop processes messages from the queue
func (b *Broadcaster) broadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second) // Adjust timing as needed
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[Broadcaster] [broadcastLoop] Broadcasting loop stopped due to context cancellation")
			return
		case <-ticker.C:
			err := b.broadcastMsgs(ctx)
			if err != nil {
				log.Error().Err(err).Msg("[Broadcaster] [broadcastLoop] Failed to broadcast messages")
			}
		}
	}
}

// type queueMsg struct {
// 	Msg types.Msg
// }

// func (m *queueMsg) Bytes() []byte {
// 	b, err := json.Marshal(m)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return b
// }

// type Worker struct {
// 	sync.Mutex
// 	network    *cosmos.NetworkClient
// 	taskQueue  []core.TaskMessage
// 	buffers    []types.Msg
// 	lastActive time.Time
// 	capacity   int
// 	period     time.Duration
// 	count      int
// 	head       int
// 	tail       int
// 	exit       chan struct{}
// 	logger     queue.Logger
// 	stopOnce   sync.Once
// 	stopFlag   int32
// }

// // Run to execute new task
// func (s *Worker) Run(ctx context.Context, task core.TaskMessage) error {
// 	var queuedMsg queueMsg
// 	err := json.Unmarshal(task.Payload(), &queuedMsg)
// 	if err != nil {
// 		log.Error().Err(err).Msg("Failed to unmarshal task")
// 		return err
// 	}
// 	log.Debug().Msgf("[Worker] [Run] received message %v", queuedMsg.Msg)
// 	s.buffers = append(s.buffers, queuedMsg.Msg)
// 	if len(s.buffers) >= s.capacity || time.Since(s.lastActive) >= s.period {
// 		// Broadcast the buffers to the scalar network
// 		log.Debug().Msgf("[Worker] [Run] broadcasting %d messages", len(s.buffers))
// 		resp, err := s.network.SignAndBroadcastMsgs(ctx, s.buffers...)
// 		if err != nil {
// 			return err
// 		} else {
// 			s.buffers = nil
// 			s.lastActive = time.Now()
// 			if resp != nil && resp.Code != 0 {
// 				log.Error().Msgf("[ScalarClient] [ConfirmEvmTxs] error from network client: %v", resp.RawLog)
// 				return fmt.Errorf("error from network client: %v", resp.RawLog)
// 			} else {
// 				log.Debug().Msgf("[ScalarClient] [ConfirmEvmTxs] success broadcast confirmation txs with tx hash: %s", resp.TxHash)
// 				return nil
// 			}
// 		}
// 	}
// 	return nil
// }

// // Shutdown the worker
// func (s *Worker) Shutdown() error {
// 	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
// 		return queue.ErrQueueShutdown
// 	}

// 	s.stopOnce.Do(func() {
// 		s.Lock()
// 		count := s.count
// 		s.Unlock()
// 		if count > 0 {
// 			<-s.exit
// 		}
// 	})
// 	return nil
// }

// // Queue send task to the buffer channel
// func (s *Worker) Queue(task core.TaskMessage) error { //nolint:stylecheck
// 	if atomic.LoadInt32(&s.stopFlag) == 1 {
// 		return queue.ErrQueueShutdown
// 	}
// 	if s.capacity > 0 && s.count >= s.capacity {
// 		return queue.ErrMaxCapacity
// 	}

// 	s.Lock()
// 	if s.count == len(s.taskQueue) {
// 		s.resize(s.count * 2)
// 	}
// 	s.taskQueue[s.tail] = task
// 	s.tail = (s.tail + 1) % len(s.taskQueue)
// 	s.count++
// 	s.Unlock()

// 	return nil
// }

// // Request a new task from channel
// func (s *Worker) Request() (core.TaskMessage, error) {
// 	if atomic.LoadInt32(&s.stopFlag) == 1 && s.count == 0 {
// 		select {
// 		case s.exit <- struct{}{}:
// 		default:
// 		}
// 		return nil, queue.ErrQueueHasBeenClosed
// 	}

// 	s.Lock()
// 	defer s.Unlock()
// 	if s.count == 0 {
// 		return nil, queue.ErrNoTaskInQueue
// 	}
// 	data := s.taskQueue[s.head]
// 	s.taskQueue[s.head] = nil
// 	s.head = (s.head + 1) % len(s.taskQueue)
// 	s.count--

// 	if n := len(s.taskQueue) / 2; n > 2 && s.count <= n {
// 		s.resize(n)
// 	}

// 	return data, nil
// }

// func (q *Worker) resize(n int) {
// 	nodes := make([]core.TaskMessage, n)
// 	if q.head < q.tail {
// 		copy(nodes, q.taskQueue[q.head:q.tail])
// 	} else {
// 		copy(nodes, q.taskQueue[q.head:])
// 		copy(nodes[len(q.taskQueue)-q.head:], q.taskQueue[:q.tail])
// 	}

// 	q.tail = q.count % n
// 	q.head = 0
// 	q.taskQueue = nodes
// }

// // NewRing for create new Ring instance
// func NewWorker(network *cosmos.NetworkClient, period time.Duration, capacity int) *Worker {
// 	w := &Worker{
// 		network:    network,
// 		taskQueue:  make([]core.TaskMessage, 2),
// 		capacity:   capacity,
// 		period:     period,
// 		lastActive: time.Now(),
// 		exit:       make(chan struct{}),
// 		logger:     zerolog.New(),
// 	}

// 	return w
// }
