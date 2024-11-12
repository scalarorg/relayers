package scalar

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func decodeBase64(str string) string {
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", decoded)
}

func removeQuote(str string) string {
	return strings.Trim(str, "\"'")
}

func ParseContractCallApprovedEvent(event map[string][]string) (*IBCEvent[ContractCallApproved], error) {
	key := "axelar.evm.v1beta1.ContractCallApproved"
	eventID := removeQuote(event[key+".event_id"][0])
	hash := strings.Split(eventID, "-")[0]

	data := ContractCallApproved{
		MessageID:        eventID,
		Sender:           removeQuote(event[key+".sender"][0]),
		SourceChain:      removeQuote(event[key+".chain"][0]),
		DestinationChain: removeQuote(event[key+".destination_chain"][0]),
		ContractAddress:  removeQuote(event[key+".contract_address"][0]),
		Payload:          "", //Payload will be get from RelayData.CallContract.Payload with filter by eventID
		PayloadHash:      "0x" + decodeBase64(removeQuote(event[key+".payload_hash"][0])),
	}

	return &IBCEvent[ContractCallApproved]{
		Hash:        hash,
		SrcChannel:  event["write_acknowledgement.packet_src_channel"][0],
		DestChannel: event["write_acknowledgement.packet_dst_channel"][0],
		Args:        data,
	}, nil
}

func ParseEvmEventCompletedEvent(event map[string][]string) (*IBCEvent[string], error) {
	eventID := removeQuote(event["axelar.evm.v1beta1.EVMEventCompleted.event_id"][0])

	return &IBCEvent[string]{
		Hash:        event["tx.hash"][0],
		SrcChannel:  event["write_acknowledgement.packet_src_channel"][0],
		DestChannel: event["write_acknowledgement.packet_dst_channel"][0],
		Args:        eventID,
	}, nil
}

func ParseContractCallSubmittedEvent(event map[string][]string) (*IBCEvent[ContractCallSubmitted], error) {
	key := "axelar.axelarnet.v1beta1.ContractCallSubmitted"
	data := ContractCallSubmitted{
		MessageID:        removeQuote(event[key+".message_id"][0]),
		Sender:           removeQuote(event[key+".sender"][0]),
		SourceChain:      removeQuote(event[key+".source_chain"][0]),
		DestinationChain: removeQuote(event[key+".destination_chain"][0]),
		ContractAddress:  removeQuote(event[key+".contract_address"][0]),
		Payload:          "0x" + decodeBase64(removeQuote(event[key+".payload"][0])),
		PayloadHash:      "0x" + decodeBase64(removeQuote(event[key+".payload_hash"][0])),
	}

	return &IBCEvent[ContractCallSubmitted]{
		Hash:        event["tx.hash"][0],
		SrcChannel:  event["write_acknowledgement.packet_src_channel"][0],
		DestChannel: event["write_acknowledgement.packet_dst_channel"][0],
		Args:        data,
	}, nil
}

func ParseContractCallWithTokenSubmittedEvent(event map[string][]string) (*IBCEvent[ContractCallWithTokenSubmitted], error) {
	key := "axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted"
	var asset struct {
		Amount string `json:"amount"`
		Denom  string `json:"denom"`
	}

	data := ContractCallWithTokenSubmitted{
		MessageID:        removeQuote(event[key+".message_id"][0]),
		Sender:           removeQuote(event[key+".sender"][0]),
		SourceChain:      removeQuote(event[key+".source_chain"][0]),
		DestinationChain: removeQuote(event[key+".destination_chain"][0]),
		ContractAddress:  removeQuote(event[key+".contract_address"][0]),
		Payload:          "0x" + decodeBase64(removeQuote(event[key+".payload"][0])),
		PayloadHash:      "0x" + decodeBase64(removeQuote(event[key+".payload_hash"][0])),
	}

	err := json.Unmarshal([]byte(event[key+".asset"][0]), &asset)
	if err == nil {
		data.Amount = asset.Amount
		data.Symbol = asset.Denom
	}
	return &IBCEvent[ContractCallWithTokenSubmitted]{
		Hash:        event["tx.hash"][0],
		SrcChannel:  event["write_acknowledgement.packet_src_channel"][0],
		DestChannel: event["write_acknowledgement.packet_dst_channel"][0],
		Args:        data,
	}, nil
}

func ParseExecuteMessageEvent(event map[string][]string) (*IBCPacketEvent, error) {
	packetData := event["send_packet.packet_data"][0]
	if packetData == "" {
		return nil, fmt.Errorf("packet_data not found")
	}

	var packetDataStruct struct {
		Amount string `json:"amount"`
		Denom  string `json:"denom"`
		Memo   string `json:"memo"`
	}
	err := json.Unmarshal([]byte(packetData), &packetDataStruct)
	if err != nil {
		return nil, err
	}

	sequence, err := strconv.ParseUint(event["send_packet.packet_sequence"][0], 10, 64)
	if err != nil {
		return nil, err
	}

	data := IBCPacketEvent{
		Sequence:    int(sequence),
		Amount:      packetDataStruct.Amount,
		Denom:       packetDataStruct.Denom,
		DestChannel: event["send_packet.packet_dst_channel"][0],
		SrcChannel:  event["send_packet.packet_src_channel"][0],
		Hash:        event["tx.hash"][0],
		Memo:        packetDataStruct.Memo,
	}

	return &data, nil
}
