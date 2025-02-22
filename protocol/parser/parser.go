package parser

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	PARSE_PARAMS = 0
	PARSE_RESULT = 1
)

var ValueNotSetError = sdkerrors.New("Value Not Set ", 6662, "when trying to parse, the value that we attempted to parse did not exist")

type RPCInput interface {
	GetParams() interface{}
	GetResult() json.RawMessage
	ParseBlock(block string) (int64, error)
	GetHeaders() []pairingtypes.Metadata
}

func ParseDefaultBlockParameter(block string) (int64, error) {
	switch block {
	case "latest":
		return spectypes.LATEST_BLOCK, nil
	case "earliest":
		return spectypes.EARLIEST_BLOCK, nil
	case "pending":
		return spectypes.PENDING_BLOCK, nil
	case "safe":
		return spectypes.SAFE_BLOCK, nil
	case "finalized":
		return spectypes.FINALIZED_BLOCK, nil
	default:
		// try to parse a number
	}
	blockNum, err := strconv.ParseInt(block, 0, 64)
	if err != nil {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("invalid block value, could not parse block %s, error: %s", block, err)
	}
	if blockNum < 0 {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("invalid block value, block value was negative %d", blockNum)
	}
	return blockNum, nil
}

// this function returns the block that was requested,
func Parse(rpcInput RPCInput, blockParser spectypes.BlockParser, dataSource int) ([]interface{}, error) {
	var retval []interface{}
	var err error

	switch blockParser.ParserFunc {
	case spectypes.PARSER_FUNC_EMPTY:
		return nil, nil
	case spectypes.PARSER_FUNC_PARSE_BY_ARG:
		retval, err = ParseByArg(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_CANONICAL:
		retval, err = ParseCanonical(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY:
		retval, err = ParseDictionary(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED:
		retval, err = ParseDictionaryOrOrdered(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_DEFAULT:
		retval = ParseDefault(rpcInput, blockParser.ParserArg, dataSource)
	default:
		return nil, fmt.Errorf("unsupported block parser parserFunc")
	}

	if err != nil {
		if ValueNotSetError.Is(err) && blockParser.DefaultValue != "" {
			// means this parsing failed because the value did not exist on an optional param
			retval = appendInterfaceToInterfaceArray(blockParser.DefaultValue)
		} else {
			return nil, err
		}
	}

	return retval, nil
}

func ParseDefault(rpcInput RPCInput, input []string, dataSource int) []interface{} {
	retArr := make([]interface{}, 0)
	retArr = append(retArr, input[0])
	return retArr
}

// this function returns the block that was requested,
func ParseBlockFromParams(rpcInput RPCInput, blockParser spectypes.BlockParser) (int64, error) {
	result, err := Parse(rpcInput, blockParser, PARSE_PARAMS)
	if err != nil || result == nil {
		return spectypes.NOT_APPLICABLE, err
	}
	resString, ok := result[0].(string)
	if !ok {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("ParseBlockFromParams - result[0].(string) - type assertion failed, type:" + fmt.Sprintf("%s", result[0]))
	}
	return rpcInput.ParseBlock(resString)
}

func ParseFromReply(rpcInput RPCInput, blockParser spectypes.BlockParser) (string, error) {
	result, err := Parse(rpcInput, blockParser, PARSE_RESULT)
	if err != nil || result == nil {
		return "", err
	}

	response, ok := result[0].(string)
	if !ok {
		return "", errors.New("result is not string parseable")
	}

	if strings.Contains(response, "\"") {
		response, err = strconv.Unquote(response)
		if err != nil {
			return "", err
		}
	}

	return response, nil
}

// this function returns the block that was requested,
func ParseBlockFromReply(rpcInput RPCInput, blockParser spectypes.BlockParser) (int64, error) {
	result, err := ParseFromReply(rpcInput, blockParser)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}
	return rpcInput.ParseBlock(result)
}

// this function returns the block that was requested,
func ParseMessageResponse(rpcInput RPCInput, resultParser spectypes.BlockParser) (string, error) {
	parsedResults, err := Parse(rpcInput, resultParser, PARSE_RESULT)
	if err != nil {
		return "", err
	}
	rawResult, ok := parsedResults[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", utils.LavaFormatError("Failed to Convert blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)", nil, utils.Attribute{Key: "blockData", Value: parsedResults[spectypes.DEFAULT_PARSED_RESULT_INDEX]})
	}
	return parseResponseByEncoding([]byte(rawResult), resultParser.Encoding)
}

// align hash encoding to base64 string, to save up on space and allow comparisons
func parseResponseByEncoding(rawResult []byte, encoding string) (string, error) {
	switch encoding {
	case spectypes.EncodingBase64:
		return string(rawResult), nil
	case spectypes.EncodingHex:
		hexString := strings.TrimPrefix(string(rawResult), "0x") // some protocols return 0x in their hex responses
		if len(hexString)%2 != 0 {
			// some hashes are hex but can't be encoded as base 64 without passing
			hexString = "0" + hexString
		}
		hexBytes, err := hex.DecodeString(hexString)
		if err != nil {
			return "", utils.LavaFormatError("tried decoding a hex response in parseResponseByEncoding but failed", err, utils.Attribute{Key: "data", Value: hexString})
		}
		return base64.StdEncoding.EncodeToString(hexBytes), nil
	default:
		return string(rawResult), nil
	}
}

// Move to RPCInput
func GetDataToParse(rpcInput RPCInput, dataSource int) (interface{}, error) {
	switch dataSource {
	case PARSE_PARAMS:
		return rpcInput.GetParams(), nil
	case PARSE_RESULT:
		interfaceArr := []interface{}{}
		var data map[string]interface{}
		unmarshalled := rpcInput.GetResult()
		if len(unmarshalled) == 0 {
			return nil, fmt.Errorf("GetDataToParse failure Get.Result is empty")
		}
		// Try to unmarshal and if the data is unmarshalable then return the data itself
		err := json.Unmarshal(unmarshalled, &data)
		if err != nil {
			interfaceArr = append(interfaceArr, unmarshalled)
		} else {
			interfaceArr = append(interfaceArr, data)
		}

		return interfaceArr, nil
	default:
		return nil, fmt.Errorf("unsupported block parser parserFunc")
	}
}

func blockInterfaceToString(block interface{}) string {
	switch castedBlock := block.(type) {
	case string:
		return castedBlock
	case float64:
		return strconv.FormatFloat(castedBlock, 'f', -1, 64)

	case int64:
		return strconv.FormatInt(castedBlock, 10)
	case uint64:
		return strconv.FormatUint(castedBlock, 10)
	default:
		return fmt.Sprintf("%s", block)
	}
}

func ParseByArg(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// specified block is one of the direct parameters, input should be one string defining the location of the block
	if len(input) != 1 {
		return nil, utils.LavaFormatProduction("invalid input format, input length", nil, utils.Attribute{Key: "input_len", Value: strconv.Itoa(len(input))})
	}
	inp := input[0]
	param_index, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return nil, utils.LavaFormatProduction("invalid input format, input isn't an unsigned index", err, utils.Attribute{Key: "input", Value: inp})
	}

	unmarshalledData, err := GetDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, utils.LavaFormatProduction("invalid input format, data is not json", err, utils.Attribute{Key: "data", Value: unmarshalledData})
	}
	switch unmarshaledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		if uint64(len(unmarshaledDataTyped)) <= param_index {
			return nil, ValueNotSetError
		}
		block := unmarshaledDataTyped[param_index]
		// TODO: turn this into type assertion instead

		retArr := make([]interface{}, 0)
		retArr = append(retArr, blockInterfaceToString(block))
		return retArr, nil
	default:
		// Parse by arg can be only list as we dont have the name of the height property.
		return nil, utils.LavaFormatError("Parse type unsupported in parse by arg, only list parameters are currently supported", nil, utils.Attribute{Key: "request", Value: unmarshaledDataTyped})
	}
}

// expect input to be keys[a,b,c] and a canonical object such as
//
//	{
//	  "a": {
//	      "b": {
//	         "c": "wanted result"
//	       }
//	   }
//	}
//
// should output an interface array with "wanted result" in first index 0
func ParseCanonical(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	unmarshalledData, err := GetDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	switch unmarshaledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		inp := input[0]
		param_index, err := strconv.ParseUint(inp, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
		}
		if uint64(len(unmarshaledDataTyped)) <= param_index {
			return nil, ValueNotSetError
		}
		blockContainer := unmarshaledDataTyped[param_index]
		for _, key := range input[1:] {
			// type assertion for blockcontainer
			if blockContainer, ok := blockContainer.(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid parser input format, blockContainer is %v and not map[string]interface{} and tried to get a field inside: %s, unmarshaledDataTyped: %s", blockContainer, key, unmarshaledDataTyped)
			}

			// assertion for key
			if container, ok := blockContainer.(map[string]interface{})[key]; ok {
				blockContainer = container
			} else {
				return nil, fmt.Errorf("invalid input format, blockContainer %s does not have field inside: %s, unmarshaledDataTyped: %s", blockContainer, key, unmarshaledDataTyped)
			}
		}
		retArr := make([]interface{}, 0)
		retArr = append(retArr, blockInterfaceToString(blockContainer))
		return retArr, nil
	case map[string]interface{}:
		inp := input[0]
		_, err := strconv.ParseUint(inp, 10, 32)
		var relevantInput []string
		if err == nil {
			relevantInput = input[1:]
		} else {
			relevantInput = input
		}
		for idx, key := range relevantInput {
			if val, ok := unmarshaledDataTyped[key]; ok {
				if idx == (len(relevantInput) - 1) {
					retArr := make([]interface{}, 0)
					retArr = append(retArr, blockInterfaceToString(val))
					return retArr, nil
				}
				// if we didn't get to the last elemnt continue deeper by chaning unmarshaledDataTyped
				switch v := val.(type) {
				case map[string]interface{}:
					unmarshaledDataTyped = v
				default:
					return nil, fmt.Errorf("failed to parse, %s is not of type map[string]interface{} \nmore information: %s", v, unmarshalledData)
				}
			} else {
				return nil, ValueNotSetError
			}
		}
	default:
		// Parse by arg can be only list as we dont have the name of the height property.
		return nil, fmt.Errorf("not Supported ParseCanonical with other types %s", unmarshaledDataTyped)
	}
	return nil, fmt.Errorf("should not get here, parsing failed %s", unmarshalledData)
}

// ParseDictionary return a value of prop specified in args if exists in dictionary
// if not return an error
func ParseDictionary(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// Validate number of arguments
	// The number of arguments should be 2
	// [prop_name,separator]
	if len(input) != 2 {
		return nil, fmt.Errorf("invalid input format, input length: %d and needs to be 2", len(input))
	}

	// Unmarshall data
	unmarshalledData, err := GetDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	// Extract arguments
	propName := input[0]
	innerSeparator := input[1]

	switch unmarshalledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		// If value attribute with propName exists in array return it
		value := parseArrayOfInterfaces(unmarshalledDataTyped, propName, innerSeparator)
		if value != nil {
			return value, nil
		}

		// Else return an error
		return nil, ValueNotSetError
	case map[string]interface{}:
		// If attribute with key propName exists return value
		if val, ok := unmarshalledDataTyped[propName]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// Else return an error
		return nil, ValueNotSetError
	default:
		return nil, fmt.Errorf("not Supported ParseDictionary with other types: %T", unmarshalledData)
	}
}

// ParseDictionaryOrOrdered return a value of prop specified in args if exists in dictionary
// if not return an item from specified index
func ParseDictionaryOrOrdered(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// Validate number of arguments
	// The number of arguments should be 3
	// [prop_name,separator,parameter order if not found]
	if len(input) != 3 {
		return nil, fmt.Errorf("ParseDictionaryOrOrdered: invalid input format, input length: %d and needs to be 3: %s", len(input), strings.Join(input, ","))
	}

	// Unmarshall data
	unmarshalledData, err := GetDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	// Extract arguments
	propName := input[0]
	innerSeparator := input[1]
	inp := input[2]

	// Convert prop index to the uint
	propIndex, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
	}

	switch unmarshalledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		// If value attribute with propName exists in array return it
		value := parseArrayOfInterfaces(unmarshalledDataTyped, propName, innerSeparator)
		if value != nil {
			return value, nil
		}

		// If not make sure there are enough elements
		if uint64(len(unmarshalledDataTyped)) <= propIndex {
			return nil, ValueNotSetError
		}

		// Fetch value using prop index
		block := unmarshalledDataTyped[propIndex]
		return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(block))
	case map[string]interface{}:
		// If attribute with key propName exists return value
		if val, ok := unmarshalledDataTyped[propName]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// If attribute with key index exists return value
		if val, ok := unmarshalledDataTyped[inp]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// Else return not set error
		return nil, ValueNotSetError
	default:
		return nil, fmt.Errorf("not Supported ParseDictionary with other types: %T", unmarshalledData)
	}
}

// parseArrayOfInterfaces returns value of item with specified prop name
// If it doesn't exist return nil
func parseArrayOfInterfaces(data []interface{}, propName, innerSeparator string) []interface{} {
	// Iterate over unmarshalled data
	for _, val := range data {
		if prop, ok := val.(string); ok {
			// split the value by innerSeparator
			valueArr := strings.SplitN(prop, innerSeparator, 2)

			// Check if the name match prop name
			if valueArr[0] != propName || len(valueArr) < 2 {
				// if not continue
				continue
			} else {
				// if yes return the value
				return appendInterfaceToInterfaceArray(valueArr[1])
			}
		}
	}

	return nil
}

// appendInterfaceToInterfaceArray appends interface to interface array
func appendInterfaceToInterfaceArray(value interface{}) []interface{} {
	retArr := make([]interface{}, 0)
	retArr = append(retArr, value)
	return retArr
}

// appendInterfaceToInterfaceArrayWithError appends interface to interface array
// returns a valueNotSetError if the value is an empty string
func appendInterfaceToInterfaceArrayWithError(value string) ([]interface{}, error) {
	if value == "" || value == "0" || value == "%!s(<nil>)" {
		return nil, ValueNotSetError
	}
	return appendInterfaceToInterfaceArray(value), nil
}
