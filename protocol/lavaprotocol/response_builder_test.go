package lavaprotocol

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func unresponsiveProviderStub() []*pairingtypes.ReportedProvider {
	return []*pairingtypes.ReportedProvider{{Address: "stub"}}
}

func TestSignAndExtractResponse(t *testing.T) {
	ctx := context.Background()
	// consumer
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	// provider
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:                       20,
		LatestRelayCu:               10, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   123,
		Client:                      nil,
		RelayNum:                    1,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 55, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.Nil(t, err)

	// check signature
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
	finalizedBlockHashes := map[int64]interface{}{123: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = 123
	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply, true)
	require.NoError(t, err)
	err = VerifyRelayReply(reply, relay, provider_address.String())
	require.NoError(t, err)
	_, _, err = VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)
}

func TestSignAndExtractResponseLatest(t *testing.T) {
	ctx := context.Background()
	// consumer
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	// provider
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	testSpecId := "BLAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:                       20,
		LatestRelayCu:               10, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   123,
		Client:                      nil,
		RelayNum:                    1,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "banana",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", testSpecId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
	latestBlock := int64(123)
	// provider handling the response
	finalizedBlockHashes := map[int64]interface{}{latestBlock: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = latestBlock
	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply, true)
	require.NoError(t, err)
	err = VerifyRelayReply(reply, relay, provider_address.String())
	require.NoError(t, err)
	_, _, err = VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)
}
