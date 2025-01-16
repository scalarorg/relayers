package scalar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/scalar-core/x/chains/types"
	//gogoproto "google.golang.org/protobuf/proto"
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
	return strings.Trim(str, "\"")
}

//	func ParseEvent[T proto.Message](rawData map[string][]string, messages T) ([]T, error) {
//		msgName, values, err := parseTarget(rawData, messages)
//		if err != nil {
//			return nil, fmt.Errorf("failed to parse type: %T", reflect.TypeOf(messages))
//		}
//		log.Info().Str("MessageType", msgName).Msg("Successful parsed event")
//		// // Handle both struct and pointer cases
//		// if targetVal.Kind() != reflect.Struct && targetVal.Kind() != reflect.Ptr && targetVal.Elem().Kind() != reflect.Struct {
//		// 	return fmt.Errorf("target must be a struct or pointer to struct, got: %T", target)
//		// }
//		return values, nil
//	}
func ParseIBCEvent[T proto.Message](rawData map[string][]string) ([]IBCEvent[T], error) {
	jsonDatas := []map[string]string{}
	var args T
	msgType := reflect.TypeOf(args).Elem()
	msg := reflect.New(msgType).Interface().(T)
	msgName := proto.MessageName(msg)
	nameLen := len(msgName)
	for key, values := range rawData {
		if strings.HasPrefix(key, msgName) && key[nameLen] == '.' {
			fieldName := key[nameLen+1:]
			for i, value := range values {
				if len(jsonDatas) <= i {
					jsonDatas = append(jsonDatas, map[string]string{})
				}
				jsonData := jsonDatas[i]
				jsonData[fieldName] = value
				jsonDatas[i] = jsonData
			}
		}
	}
	result := []IBCEvent[T]{}
	for _, data := range jsonDatas {
		instance := copyMessage(msg)
		log.Debug().Any("JsonData", data).Msgf("Try to parser to type %T", msg)
		err := UnmarshalJson(data, instance)
		if err != nil {
			log.Error().Err(err).Any("JsonData", data).Msg("Cannot unmarshal message")
		} else {
			log.Debug().Any("Instance", instance).Msg("Parse event successfull")
		}
		args := instance.(T)
		result = append(result, IBCEvent[T]{
			Args: args,
		})
	}
	return result, nil
}

func parseTarget[T proto.Message](rawData map[string][]string, msg T) (string, []T, error) {
	jsonDatas := []map[string]string{}
	msgName := proto.MessageName(msg)
	fmt.Printf("MsgName %s", msgName)
	nameLen := len(msgName) + 1
	//msgType := reflect.TypeOf(msg)
	for key, values := range rawData {
		if strings.HasPrefix(key, msgName) {
			fieldName := key[nameLen:]
			for i, value := range values {
				if len(jsonDatas) <= i {
					jsonDatas = append(jsonDatas, map[string]string{})
				}
				jsonData := jsonDatas[i]
				jsonData[fieldName] = value
				jsonDatas[i] = jsonData
			}
		}
	}
	result := []T{}
	for _, data := range jsonDatas {
		// Convert map back to JSON
		// jsonData, err := json.Marshal(data)
		// if err != nil {
		// 	log.Error().Err(err).Any("JsonData", data).Msg("Parse error")
		// }
		// Unmarshal JSON into the target struct
		// err = proto.Unmarshal(jsonData, instance)
		//err = gogoproto.Unmarshal(jsonData, instance)
		//err = json.Unmarshal(jsonData, instance)
		instance := copyMessage(msg)
		err := bindValue(data, instance)

		if err != nil {
			log.Error().Err(err).Any("JsonData", data).Msg("Cannot unmarshal message")
		}
		// log.Info().Any("JsonData", data).Any("Instance", instance).Msg("BindValues")
		result = append(result, instance.(T))
	}
	return msgName, result, nil

}
func copyMessage(inst proto.Message) proto.Message {
	typ := reflect.TypeOf(inst)
	val := reflect.ValueOf(inst)

	if val.Kind() == reflect.Pointer {
		vp := reflect.New(typ.Elem())
		return vp.Interface().(proto.Message)
	} else {
		v := reflect.Zero(typ)
		return v.Interface().(proto.Message)
	}
}
func bindValue(data map[string]string, target interface{}) error {
	targetVal := reflect.ValueOf(target).Elem()
	targetType := targetVal.Type()
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := targetVal.Field(i)

		tag := getJSONTag(field)
		if value, exists := data[tag]; exists {
			if err := setFieldValue(fieldValue, value, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

// getJSONTag extracts the JSON tag name or defaults to the field name.
func getJSONTag(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return field.Name
	}
	return strings.Split(tag, ",")[0]
}

// setFieldValue sets the appropriate value for a struct field.
func setFieldValue(fieldValue reflect.Value, value string, tag string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}
		fieldValue.SetString(value)
	case reflect.Slice:
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			return setByteArrayValue(fieldValue, value, tag)
		}
		return fmt.Errorf("unsupported slice type for field '%s'", tag)
	case reflect.Array:
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			return setArrayValue(fieldValue, value, tag)
		}
		return fmt.Errorf("unsupported array type for field '%s'", tag)
	default:
		return fmt.Errorf("unsupported field type %s for field '%s'", fieldValue.Kind(), tag)
	}
	return nil
}

// setByteArrayValue handles slices and decodes from either JSON array notation or hex string.
func setByteArrayValue(fieldValue reflect.Value, value string, tag string) error {
	var byteArray []byte
	var err error

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		byteArray, err = parseByteArray(value)
	} else {
		byteArray, err = hex.DecodeString(value)
	}

	if err != nil {
		return fmt.Errorf("failed to parse byte array for field '%s': %w", tag, err)
	}

	fieldValue.SetBytes(byteArray)
	return nil
}

// setArrayValue handles fixed-size arrays and decodes from JSON array notation or hex string.
func setArrayValue(fieldValue reflect.Value, value string, tag string) error {
	var byteArray []byte
	var err error

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		byteArray, err = parseByteArray(value)
	} else {
		byteArray, err = hex.DecodeString(value)
	}

	if err != nil {
		return fmt.Errorf("failed to parse array for field '%s': %w", tag, err)
	}

	reflect.Copy(fieldValue, reflect.ValueOf(byteArray))
	return nil
}

// parseByteArray converts a JSON array-like string into a byte slice.
func parseByteArray(value string) ([]byte, error) {
	value = strings.Trim(value, "[]")
	parts := strings.Split(value, ",")
	byteArray := make([]byte, len(parts))

	for i, part := range parts {
		num, err := parseByte(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		byteArray[i] = num
	}
	return byteArray, nil
}

// parseByte parses a single byte value from a string.
func parseByte(value string) (byte, error) {
	var num int
	_, err := fmt.Sscanf(value, "%d", &num)
	if err != nil {
		return 0, err
	}
	if num < 0 || num > 255 {
		return 0, fmt.Errorf("value out of range: %d", num)
	}
	return byte(num), nil
}

func ParseTokenSentEvent(event map[string][]string) ([]IBCEvent[*types.EventTokenSent], error) {
	return nil, nil
}

// func ParseContractCallApprovedEvent(event map[string][]string) ([]IBCEvent[*ContractCallApproved], error) {
// 	log.Debug().Msgf("[ScalarClient] [ParseContractCallApprovedEvent] start parser")
// 	key := EventTypeContractCallApproved
// 	eventIds := event[key+".event_id"]
// 	senders := event[key+".sender"]
// 	sourceChains := event[key+".chain"]
// 	destinationChains := event[key+".destination_chain"]
// 	contractAddresses := event[key+".contract_address"]
// 	commandIds := event[key+".command_id"]
// 	payloadHashes := event[key+".payload_hash"]
// 	srcChannels := event["write_acknowledgement.packet_src_channel"]
// 	destChannels := event["write_acknowledgement.packet_dst_channel"]
// 	events := make([]IBCEvent[*types.ContractCallApproved], len(eventIds))
// 	for ind, eventId := range eventIds {
// 		eventID := removeQuote(eventId)
// 		hash := strings.Split(eventID, "-")[0]
// 		payloadHash, err := DecodeIntArrayToHexString(payloadHashes[ind])
// 		if err != nil {
// 			log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", payloadHashes[ind], err)
// 		}
// 		commandIDHex, err := DecodeIntArrayToHexString(commandIds[ind])
// 		if err != nil {
// 			log.Warn().Msgf("Failed to decode command ID: %v, error: %v", commandIds[ind], err)
// 		}
// 		commandID, err := types.HexToCommandID(commandIDHex)
// 		data := &types.ContractCallApproved{
// 			EventID:          types.EventID(eventID),
// 			Sender:           removeQuote(senders[ind]),
// 			Chain:            exported.ChainName(removeQuote(sourceChains[ind])),
// 			DestinationChain: exported.ChainName(removeQuote(destinationChains[ind])),
// 			ContractAddress:  removeQuote(contractAddresses[ind]),
// 			//Payload:          "", //Payload will be get from RelayData.CallContract.Payload with filter by eventID
// 			PayloadHash: types.Hash(common.HexToHash("0x" + payloadHash)),
// 			CommandID:   commandID,
// 		}
// 		var srcChannel string
// 		var destChannel string
// 		if len(srcChannels) > ind {
// 			srcChannel = srcChannels[ind]
// 		}
// 		if len(destChannels) > ind {
// 			destChannel = destChannels[ind]
// 		}
// 		events[ind] = IBCEvent[*types.ContractCallApproved]{
// 			Hash:        hash,
// 			SrcChannel:  srcChannel,
// 			DestChannel: destChannel,
// 			Args:        data,
// 		}
// 	}
// 	log.Debug().Msgf("[ScalarClient] [ParseContractCallApprovedEvent] parsed events: %v", events)
// 	return events, nil
// }

//	func ParseEvmEventCompletedEvent(event map[string][]string) ([]IBCEvent[EVMEventCompleted], error) {
//		log.Debug().Msgf("[ScalarClient] [ParseEvmEventCompletedEvent] start parser")
//		eventIds := event[EventTypeEVMEventCompleted+".event_id"]
//		txHashs := event["tx.hash"]
//		srcChannels := event["write_acknowledgement.packet_src_channel"]
//		destChannels := event["write_acknowledgement.packet_dst_channel"]
//		events := make([]IBCEvent[EVMEventCompleted], len(eventIds))
//		for ind, eventId := range eventIds {
//			eventID := removeQuote(eventId)
//			chainEventCompleted := ChainEventCompleted {
//				Chain: "",
//				EventId: "",
//				Type: ""
//			}
//			args := EVMEventCompleted{
//				ChainEventCompleted: ,
//				ID:      eventID,
//				Payload: "",
//			}
//			var srcChannel string
//			var destChannel string
//			if len(srcChannels) > ind {
//				srcChannel = srcChannels[ind]
//			}
//			if len(destChannels) > ind {
//				destChannel = destChannels[ind]
//			}
//			events[ind] = IBCEvent[EVMEventCompleted]{
//				Hash:        txHashs[ind],
//				SrcChannel:  srcChannel,
//				DestChannel: destChannel,
//				Args:        args,
//			}
//		}
//		log.Debug().Msgf("[ScalarClient] [ParseEvmEventCompletedEvent] parsed events: %v", events)
//		return events, nil
//	}
func ParseAllNewBlockEvent(event map[string][]string) ([]IBCEvent[any], error) {
	log.Debug().Msgf("[ScalarClient] [ParseAllNewBlockEvent] input event: %v", event)
	return nil, nil
}

// func ParseContractCallSubmittedEvent(event map[string][]string) ([]IBCEvent[ContractCallSubmitted], error) {
// 	log.Debug().Msgf("[ScalarClient] Received ContractCallSubmitted event: %d", len(event))
// 	key := "scalar.scalarnet.v1beta1.ContractCallSubmitted"
// 	messageIDs := event[key+".message_id"]
// 	txHashes := event["tx.hash"]
// 	senders := event[key+".sender"]
// 	sourceChains := event[key+".source_chain"]
// 	destinationChains := event[key+".destination_chain"]
// 	contractAddresses := event[key+".contract_address"]
// 	payloads := event[key+".payload"]
// 	payloadHashes := event[key+".payload_hash"]
// 	srcChannels := event["write_acknowledgement.packet_src_channel"]
// 	destChannels := event["write_acknowledgement.packet_dst_channel"]
// 	events := make([]IBCEvent[ContractCallSubmitted], len(messageIDs))
// 	for ind := range messageIDs {
// 		payloadHash, err := DecodeIntArrayToHexString(payloadHashes[ind])
// 		if err != nil {
// 			log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", payloadHashes[ind], err)
// 		}
// 		payload, err := DecodeIntArrayToBytes(payloads[ind])
// 		if err != nil {
// 			log.Warn().Msgf("Failed to decode payload: %v, error: %v", payloads[ind], err)
// 		}
// 		data := ContractCallSubmitted{
// 			MessageID:        removeQuote(messageIDs[ind]),
// 			Sender:           removeQuote(senders[ind]),
// 			SourceChain:      removeQuote(sourceChains[ind]),
// 			DestinationChain: removeQuote(destinationChains[ind]),
// 			ContractAddress:  removeQuote(contractAddresses[ind]),
// 			Payload:          hex.EncodeToString(payload),
// 			PayloadHash:      "0x" + payloadHash,
// 		}
// 		var srcChannel string
// 		var destChannel string
// 		if len(srcChannels) > ind {
// 			srcChannel = srcChannels[ind]
// 		}
// 		if len(destChannels) > ind {
// 			destChannel = destChannels[ind]
// 		}
// 		events[ind] = IBCEvent[ContractCallSubmitted]{
// 			Hash:        txHashes[ind],
// 			SrcChannel:  srcChannel,
// 			DestChannel: destChannel,
// 			Args:        data,
// 		}
// 	}
// 	log.Debug().Msgf("[ScalarClient] Parsed ContractCallSubmitted events: %v", events)
// 	return events, nil
// }

func ParseContractCallWithTokenSubmittedEvent(event map[string][]string) ([]IBCEvent[ContractCallWithTokenSubmitted], error) {
	log.Debug().Msgf("[ScalarClient] Received ContractCallWithTokenSubmitted event: %d", len(event))
	key := "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted"
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
