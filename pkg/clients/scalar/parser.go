package scalar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func DecodeIntArrayToBytes(input string) ([]byte, error) {
	// Parse the input string as a slice of integers
	var intArray []int
	err := json.Unmarshal([]byte(input), &intArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %v", err)
	}
	// Convert the slice of integers to a slice of bytes
	byteArray := make([]byte, len(intArray))
	for i, v := range intArray {
		byteArray[i] = byte(v)
	}
	return byteArray, nil
}
func DecodeIntArrayToHexString(input string) (string, error) {
	byteArray, err := DecodeIntArrayToBytes(input)
	if err != nil {
		return "", err
	}
	hexString := hex.EncodeToString(byteArray)
	return hexString, nil
}

func removeQuote(str string) string {
	return strings.Trim(str, "\"'")
}

func ParseContractCallApprovedEvent(event map[string][]string) ([]IBCEvent[ContractCallApproved], error) {
	log.Debug().Msgf("[ScalarClient] Received ContractCallApproved event")
	key := "axelar.evm.v1beta1.ContractCallApproved"
	eventIds := event[key+".event_id"]
	senders := event[key+".sender"]
	sourceChains := event[key+".chain"]
	destinationChains := event[key+".destination_chain"]
	contractAddresses := event[key+".contract_address"]
	commandIds := event[key+".command_id"]
	payloadHashes := event[key+".payload_hash"]
	srcChannels := event["write_acknowledgement.packet_src_channel"]
	destChannels := event["write_acknowledgement.packet_dst_channel"]
	events := make([]IBCEvent[ContractCallApproved], len(eventIds))
	for ind, eventId := range eventIds {
		eventID := removeQuote(eventId)
		hash := strings.Split(eventID, "-")[0]
		payloadHash, err := DecodeIntArrayToHexString(payloadHashes[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", payloadHashes[ind], err)
		}
		commandID, err := DecodeIntArrayToHexString(commandIds[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode command ID: %v, error: %v", commandIds[ind], err)
		}
		data := ContractCallApproved{
			MessageID:        eventID,
			Sender:           removeQuote(senders[ind]),
			SourceChain:      removeQuote(sourceChains[ind]),
			DestinationChain: removeQuote(destinationChains[ind]),
			ContractAddress:  removeQuote(contractAddresses[ind]),
			Payload:          "", //Payload will be get from RelayData.CallContract.Payload with filter by eventID
			PayloadHash:      "0x" + payloadHash,
			CommandID:        commandID,
		}
		var srcChannel string
		var destChannel string
		if len(srcChannels) > ind {
			srcChannel = srcChannels[ind]
		}
		if len(destChannels) > ind {
			destChannel = destChannels[ind]
		}
		events[ind] = IBCEvent[ContractCallApproved]{
			Hash:        hash,
			SrcChannel:  srcChannel,
			DestChannel: destChannel,
			Args:        data,
		}
	}
	log.Debug().Msgf("[ScalarClient] Parsed %d ContractCallApproved events: %v", len(events), events)
	return events, nil
}
func ParseSignCommandsEvent(event map[string][]string) ([]IBCEvent[SignCommands], error) {
	log.Debug().Msgf("[ScalarClient] Received SignCommands event: %d", len(event))
	return nil, nil
}
func ParseEvmEventCompletedEvent(event map[string][]string) ([]IBCEvent[EVMEventCompleted], error) {
	log.Debug().Msgf("[ScalarClient] Received EVMEventCompleted event: %d", len(event))
	eventIds := event["axelar.evm.v1beta1.EVMEventCompleted.event_id"]
	txHashs := event["tx.hash"]
	srcChannels := event["write_acknowledgement.packet_src_channel"]
	destChannels := event["write_acknowledgement.packet_dst_channel"]
	events := make([]IBCEvent[EVMEventCompleted], len(eventIds))
	for ind, eventId := range eventIds {
		eventID := removeQuote(eventId)
		args := EVMEventCompleted{
			ID:      eventID,
			Payload: "",
		}
		var srcChannel string
		var destChannel string
		if len(srcChannels) > ind {
			srcChannel = srcChannels[ind]
		}
		if len(destChannels) > ind {
			destChannel = destChannels[ind]
		}
		events[ind] = IBCEvent[EVMEventCompleted]{
			Hash:        txHashs[ind],
			SrcChannel:  srcChannel,
			DestChannel: destChannel,
			Args:        args,
		}
	}
	log.Debug().Msgf("[ScalarClient] Parsed EVMEventCompleted events: %v", events)
	return events, nil
}
func ParseAllEvent(event map[string][]string) ([]IBCEvent[any], error) {
	log.Debug().Msgf("[ScalarClient] Received event: %v", event)
	return nil, nil
}
func ParseContractCallSubmittedEvent(event map[string][]string) ([]IBCEvent[ContractCallSubmitted], error) {
	log.Debug().Msgf("[ScalarClient] Received ContractCallSubmitted event: %d", len(event))
	key := "axelar.axelarnet.v1beta1.ContractCallSubmitted"
	messageIDs := event[key+".message_id"]
	txHashes := event["tx.hash"]
	senders := event[key+".sender"]
	sourceChains := event[key+".source_chain"]
	destinationChains := event[key+".destination_chain"]
	contractAddresses := event[key+".contract_address"]
	payloads := event[key+".payload"]
	payloadHashes := event[key+".payload_hash"]
	srcChannels := event["write_acknowledgement.packet_src_channel"]
	destChannels := event["write_acknowledgement.packet_dst_channel"]
	events := make([]IBCEvent[ContractCallSubmitted], len(messageIDs))
	for ind := range messageIDs {
		payloadHash, err := DecodeIntArrayToHexString(payloadHashes[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", payloadHashes[ind], err)
		}
		payload, err := DecodeIntArrayToBytes(payloads[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode payload: %v, error: %v", payloads[ind], err)
		}
		data := ContractCallSubmitted{
			MessageID:        removeQuote(messageIDs[ind]),
			Sender:           removeQuote(senders[ind]),
			SourceChain:      removeQuote(sourceChains[ind]),
			DestinationChain: removeQuote(destinationChains[ind]),
			ContractAddress:  removeQuote(contractAddresses[ind]),
			Payload:          hex.EncodeToString(payload),
			PayloadHash:      "0x" + payloadHash,
		}
		var srcChannel string
		var destChannel string
		if len(srcChannels) > ind {
			srcChannel = srcChannels[ind]
		}
		if len(destChannels) > ind {
			destChannel = destChannels[ind]
		}
		events[ind] = IBCEvent[ContractCallSubmitted]{
			Hash:        txHashes[ind],
			SrcChannel:  srcChannel,
			DestChannel: destChannel,
			Args:        data,
		}
	}
	log.Debug().Msgf("[ScalarClient] Parsed ContractCallSubmitted events: %v", events)
	return events, nil
}

func ParseContractCallWithTokenSubmittedEvent(event map[string][]string) ([]IBCEvent[ContractCallWithTokenSubmitted], error) {
	log.Debug().Msgf("[ScalarClient] Received ContractCallWithTokenSubmitted event: %d", len(event))
	key := "axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted"
	messageIDs := event[key+".message_id"]
	txHashes := event["tx.hash"]
	senders := event[key+".sender"]
	sourceChains := event[key+".source_chain"]
	destinationChains := event[key+".destination_chain"]
	contractAddresses := event[key+".contract_address"]
	payloads := event[key+".payload"]
	payloadHashes := event[key+".payload_hash"]
	srcChannels := event["write_acknowledgement.packet_src_channel"]
	destChannels := event["write_acknowledgement.packet_dst_channel"]
	events := make([]IBCEvent[ContractCallWithTokenSubmitted], len(messageIDs))
	for ind := range messageIDs {
		payload, err := DecodeIntArrayToBytes(payloads[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode payload: %v, error: %v", payloads[ind], err)
		}
		payloadHash, err := DecodeIntArrayToHexString(payloadHashes[ind])
		if err != nil {
			log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", payloadHashes[ind], err)
		}
		data := ContractCallWithTokenSubmitted{
			MessageID:        removeQuote(messageIDs[ind]),
			Sender:           removeQuote(senders[ind]),
			SourceChain:      removeQuote(sourceChains[ind]),
			DestinationChain: removeQuote(destinationChains[ind]),
			ContractAddress:  removeQuote(contractAddresses[ind]),
			Payload:          "0x" + hex.EncodeToString(payload),
			PayloadHash:      "0x" + payloadHash,
		}
		var asset struct {
			Amount string `json:"amount"`
			Denom  string `json:"denom"`
		}
		err = json.Unmarshal([]byte(event[key+".asset"][ind]), &asset)
		if err == nil {
			data.Amount = asset.Amount
			data.Symbol = asset.Denom
		}
		var srcChannel string
		var destChannel string
		if len(srcChannels) > ind {
			srcChannel = srcChannels[ind]
		}
		if len(destChannels) > ind {
			destChannel = destChannels[ind]
		}
		events[ind] = IBCEvent[ContractCallWithTokenSubmitted]{
			Hash:        txHashes[ind],
			SrcChannel:  srcChannel,
			DestChannel: destChannel,
			Args:        data,
		}
	}
	log.Debug().Msgf("[ScalarClient] Parsed ContractCallWithTokenSubmitted events: %v", events)
	return events, nil
}

func ParseExecuteMessageEvent(event map[string][]string) ([]IBCPacketEvent, error) {
	log.Debug().Msgf("[ScalarClient] Received ExecuteMessage event: %d", len(event))
	packetDatas := event["send_packet.packet_data"]
	txHashes := event["tx.hash"]
	packetSequences := event["send_packet.packet_sequence"]
	srcChannels := event["send_packet.packet_src_channel"]
	destChannels := event["send_packet.packet_dst_channel"]
	events := make([]IBCPacketEvent, len(packetDatas))
	for ind, packetData := range packetDatas {
		var packetDataStruct struct {
			Amount string `json:"amount"`
			Denom  string `json:"denom"`
			Memo   string `json:"memo"`
		}
		err := json.Unmarshal([]byte(packetData), &packetDataStruct)
		if err != nil {
			return nil, err
		}

		sequence, err := strconv.ParseUint(packetSequences[ind], 10, 64)
		if err != nil {
			return nil, err
		}
		var srcChannel string
		var destChannel string
		if len(srcChannels) > ind {
			srcChannel = srcChannels[ind]
		}
		if len(destChannels) > ind {
			destChannel = destChannels[ind]
		}
		events[ind] = IBCPacketEvent{
			Sequence:    int(sequence),
			Amount:      packetDataStruct.Amount,
			Denom:       packetDataStruct.Denom,
			DestChannel: destChannel,
			SrcChannel:  srcChannel,
			Hash:        txHashes[ind],
			Memo:        packetDataStruct.Memo,
		}

	}
	log.Debug().Msgf("[ScalarClient] Parsed ExecuteMessage events: %v", events)
	return events, nil
}
