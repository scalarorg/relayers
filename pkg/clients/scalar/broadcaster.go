package scalar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/scalar-core/utils"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"
)

// https://github.com/cosmos/cosmos-sdk/blob/main/client/tx/tx.go#L31

type Broadcaster struct {
	network         *cosmos.NetworkClient
	pendingCommands *PendingCommands //Keep reference to the pending commands for store sign command request tx hash
	//We keep unique sign pending commands for each chain in broadcaster's buffer
	SignPendingCommandRequests      sync.Map
	SignPendingCommandRequestsMutex sync.Mutex
	//queue          *queue.Queue
	buffers    []types.Msg
	mutex      sync.Mutex
	isRunning  bool
	period     time.Duration
	batchSize  int //Number messages to broadcast in a transaction
	cycleCount int //Log message is printed every 1000 messages
}

func NewBroadcaster(network *cosmos.NetworkClient, pendingCommands *PendingCommands, broadcastPeriod time.Duration, batchSize int) *Broadcaster {
	return &Broadcaster{
		network:         network,
		pendingCommands: pendingCommands,
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
func (b *Broadcaster) QueueMsg(msg types.Msg) error {
	b.mutex.Lock()
	if !b.isRunning {
		b.mutex.Unlock()
		return fmt.Errorf("broadcaster is not running")
	}
	b.mutex.Unlock()
	log.Debug().Msgf("[Broadcaster] [QueueMsg] enqueue message %v", msg)
	//b.queue.Queue(&queueMsg{Msg: msg})
	b.buffers = append(b.buffers, msg)
	return nil
}

func (c *Broadcaster) ConfirmEvmTxs(chainName string, txIds []string) error {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[Broadcaster] [ConfirmEvmTxs] Enqueue for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]chainstypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainstypes.Hash(common.HexToHash(txId))
	}
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
	return c.QueueMsg(msg)
}

func (c *Broadcaster) ConfirmBtcTxs(chainName string, txIds []string) error {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[Broadcaster] [ConfirmBtcTxs] Enqueue for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]chainstypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainstypes.Hash(common.HexToHash(txId))
	}
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
	return c.QueueMsg(msg)
}

func (c *Broadcaster) AddSignEvmCommandsRequest(destinationChain string) error {
	req := chainstypes.NewSignCommandsRequest(
		c.network.GetAddress(),
		destinationChain)
	return c.QueueMsg(req)
}

// Add SignPsbtCommandsRequest to buffer
// Return true if the request is added to buffer, false if the request is already in buffer
func (c *Broadcaster) AddSignPsbtCommandsRequest(destinationChain string, psbt covtypes.Psbt) bool {
	c.SignPendingCommandRequestsMutex.Lock()
	defer c.SignPendingCommandRequestsMutex.Unlock()
	_, loaded := c.SignPendingCommandRequests.Load(destinationChain)
	if loaded {
		log.Debug().Str("Chain", destinationChain).Msgf("[Broadcaster] [AddSignUpcCommandsRequest] SignCommandRequest is already in buffer. Skip adding new one")
		return false
	} else {
		req := chainstypes.NewSignPsbtCommandRequest(
			c.network.GetAddress(),
			destinationChain,
			psbt)
		err := c.QueueMsg(req)
		if err != nil {
			log.Error().Err(err).Msgf("[Broadcaster] [AddSignPsbtCommandsRequest] Failed to add SignPsbtCommandRequest to buffer")
			return false
		}
		c.SignPendingCommandRequests.Store(destinationChain, 1)
		return true
	}
}

// Add SignBtcCommandsRequest to buffer
// Return true if the request is added to buffer, false if the request is already in buffer
func (c *Broadcaster) AddSignUpcCommandsRequest(destinationChain string) bool {
	c.SignPendingCommandRequestsMutex.Lock()
	defer c.SignPendingCommandRequestsMutex.Unlock()
	_, loaded := c.SignPendingCommandRequests.Load(destinationChain)
	if loaded {
		log.Debug().Str("Chain", destinationChain).Msgf("[Broadcaster] [AddSignUpcCommandsRequest] SignCommandRequest is already in buffer. Skip adding new one")
		return false
	} else {
		req := chainstypes.NewSignBtcCommandsRequest(
			c.network.GetAddress(),
			destinationChain)

		err := c.QueueMsg(req)
		if err != nil {
			log.Error().Err(err).Msgf("[Broadcaster] [AddSignUpcCommandsRequest] Failed to add SignCommandRequest to buffer")
			return false
		}
		c.SignPendingCommandRequests.Store(destinationChain, 1)
		return true
	}
}

func (c *Broadcaster) CleanPendingCommandRequests(chain string) {
	c.SignPendingCommandRequestsMutex.Lock()
	defer c.SignPendingCommandRequestsMutex.Unlock()
	c.SignPendingCommandRequests.Delete(chain)
}

func (c *Broadcaster) CreatePendingTransfersRequest(chain string) error {
	req := chainstypes.CreatePendingTransfersRequest{
		Sender: c.network.GetAddress(),
		Chain:  nexus.ChainName(chain),
	}
	return c.QueueMsg(&req)
}
func (b *Broadcaster) pushFailedMsgBackToBuffer(msgs []types.Msg) error {
	b.mutex.Lock()
	b.buffers = append(msgs, b.buffers...)
	b.mutex.Unlock()
	log.Info().
		Int("remain_buffer_size", len(b.buffers)).
		Msg("[Broadcaster] Waiting for next broadcasting")
	return nil
}

// try broadcast fist messages in the buffer
func (b *Broadcaster) broadcastMsgs(ctx context.Context) error {
	var msgs []types.Msg
	b.mutex.Lock()
	if len(b.buffers) > b.batchSize {
		msgs = b.buffers[:b.batchSize]
		b.buffers = b.buffers[b.batchSize:]
	} else {
		msgs = b.buffers
		b.buffers = nil
	}
	b.mutex.Unlock()
	if len(msgs) == 0 {
		if b.cycleCount >= 1000 {
			log.Debug().Msg("[Broadcaster] No messages to broadcast")
			b.cycleCount = 0
		}
		return nil
	} else {
		log.Debug().Msgf("[Broadcaster] [broadcastMsgs] Broadcasting %d messages", len(msgs))
	}
	resp, err := b.network.SignAndBroadcastMsgs(ctx, msgs...)
	if err != nil {
		log.Error().Err(err).Msg("[Broadcaster] Failed to broadcast messages")
		b.pushFailedMsgBackToBuffer(msgs)
		return err
	}

	if resp.Code == 0 {
		log.Info().
			Int("msg_count", len(msgs)).
			Str("tx_hash", resp.TxHash).
			Int("remain_buffer_size", len(b.buffers)).
			Msg("[Broadcaster] Successfully broadcasted messages")
		for _, msg := range msgs {
			switch value := msg.(type) {
			case *chainstypes.SignCommandsRequest:
				log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignEvmPendingCommandsRequest")
				b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
				b.CleanPendingCommandRequests(string(value.Chain))
			case *chainstypes.SignPsbtCommandRequest:
				log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignPsbtCommandRequest")
				b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
				b.CleanPendingCommandRequests(string(value.Chain))
			case *chainstypes.SignBtcCommandsRequest:
				log.Debug().Str("Chain", string(value.Chain)).Str("TxHash", resp.TxHash).Msg("[Broadcaster] Store txHash into pending SignUpcCommandRequest")
				b.pendingCommands.StoreSignRequest(string(value.Chain), resp.TxHash)
				b.CleanPendingCommandRequests(string(value.Chain))
			default:
				log.Debug().Msgf("[Broadcaster] [successfully broadcasted]: %v of type %T", msg, msg)
			}
		}
		msgs = nil
	} else {
		log.Error().
			Uint32("code", resp.Code).
			Str("raw_log", resp.RawLog).
			Msg("[Broadcaster] Broadcast messages failed put back to the buffer")
		// put messages back to the buffer
		b.pushFailedMsgBackToBuffer(msgs)
	}
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
		default:
			if len(b.buffers) > b.batchSize {
				err := b.broadcastMsgs(ctx)
				if err != nil {
					log.Error().Err(err).Msg("[Broadcaster] [broadcastLoop] Failed to broadcast messages")
				}
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
