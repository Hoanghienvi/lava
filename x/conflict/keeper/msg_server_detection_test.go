package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	conflictconstruct "github.com/lavanet/lava/x/conflict/types/construct"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

const ProvidersCount = 5

type tester struct {
	common.Tester
	consumer  common.Account
	providers []common.Account
	plan      plantypes.Plan
	spec      spectypes.Spec
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}

	ts.AddPlan("free", common.CreateMockPlan())
	ts.AddSpec("mock", common.CreateMockSpec())

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) setupForConflict(providersCount int) *tester {
	var (
		balance int64 = 100000
		stake   int64 = 1000
	)

	ts.plan = ts.Plan("free")
	ts.spec = ts.Spec("mock")

	consumer, consumerAddr := ts.AddAccount("consumer", 0, balance)
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, ts.plan.Index, 1)
	require.Nil(ts.T, err)
	ts.consumer = consumer

	for i := 0; i < providersCount; i++ {
		providerAcct, providerAddr := ts.AddAccount(common.PROVIDER, i, balance)
		err := ts.StakeProvider(providerAddr, ts.spec, stake)
		require.Nil(ts.T, err)
		ts.providers = append(ts.providers, providerAcct)
	}

	ts.AdvanceEpoch()
	return ts
}

func TestDetection(t *testing.T) {
	ts := newTester(t)
	ts.setupForConflict(ProvidersCount)

	tests := []struct {
		name           string
		Creator        common.Account
		Provider0      common.Account
		Provider1      common.Account
		ConnectionType string
		ApiUrl         string
		BlockHeight    int64
		ChainID        string
		Data           []byte
		RequestBlock   int64
		Cusum          uint64
		RelayNum       uint64
		SeassionID     uint64
		QoSReport      *types.QualityOfServiceReport
		ReplyData      []byte
		Valid          bool
	}{
		{"HappyFlow", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 0, 100, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"CuSumChange", ts.consumer, ts.providers[0], ts.providers[2], "", "", 0, "", []byte{}, 0, 0, 100, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"RelayNumChange", ts.consumer, ts.providers[0], ts.providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"SessionIDChange", ts.consumer, ts.providers[0], ts.providers[4], "", "", 0, "", []byte{}, 0, 0, 0, 1, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"QoSNil", ts.consumer, ts.providers[2], ts.providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, nil, []byte("DIFF"), true},
		{"BadCreator", ts.providers[4], ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadConnectionType", ts.consumer, ts.providers[0], ts.providers[1], "DIFF", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadURL", ts.consumer, ts.providers[0], ts.providers[1], "", "DIFF", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadBlockHeight", ts.consumer, ts.providers[0], ts.providers[1], "", "", 10, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadChainID", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "DIFF", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadData", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte("DIFF"), 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadRequestBlock", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"SameReplyData", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _, reply, err := common.CreateMsgDetectionTest(ts.GoCtx, tt.Creator, tt.Provider0, tt.Provider1, ts.spec)
			require.Nil(t, err)

			msg.Creator = tt.Creator.Addr.String()

			// changes to request1 according to test
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.ConnectionType += tt.ConnectionType
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.ApiUrl += tt.ApiUrl
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Epoch += tt.BlockHeight
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.SpecId += tt.ChainID
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.Data = append(msg.ResponseConflict.ConflictRelayData1.Request.RelayData.Data, tt.Data...)
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.RequestBlock += tt.RequestBlock
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.CuSum += tt.Cusum
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.QosReport = tt.QoSReport
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.RelayNum += tt.RelayNum
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.SessionId += tt.SeassionID
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Provider = tt.Provider1.Addr.String()
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
			sig, err := sigs.Sign(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request.RelaySession)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = sig
			reply.Data = append(reply.Data, tt.ReplyData...)
			relayExchange := types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData1.Request, *reply)
			sig, err = sigs.Sign(tt.Provider1.SK, relayExchange)
			require.Nil(t, err)
			reply.Sig = sig
			relayFinalization := types.NewRelayFinalization(types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData1.Request, *reply), ts.consumer.Addr)
			sigBlocks, err := sigs.Sign(tt.Provider1.SK, relayFinalization)
			require.Nil(t, err)
			reply.SigBlocks = sigBlocks
			msg.ResponseConflict.ConflictRelayData1.Reply = conflictconstruct.ConstructReplyMetadata(reply, msg.ResponseConflict.ConflictRelayData1.Request)
			// send detection msg
			_, err = ts.txConflictDetection(msg)
			if tt.Valid {
				events := ts.Ctx.EventManager().Events()
				require.Nil(t, err)
				require.Equal(t, events[len(events)-1].Type, "lava_"+conflicttypes.ConflictVoteDetectionEventName)
			}
		})
	}
}
