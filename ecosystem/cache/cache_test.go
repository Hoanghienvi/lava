package cache_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/lavanet/lava/ecosystem/cache"
	"github.com/lavanet/lava/ecosystem/cache/format"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const (
	StubChainID        = "stub-chain"
	StubConnectionType = "stub-conn"
	StubApiUrl         = "stub-api"
	StubProviderAddr   = "lava@provider-stub"
	StubApiInterface   = "stub-interface"
	StubData           = "stub-data"
	StubSig            = "stub-sig"
)

func initTest() (context.Context, *cache.RelayerCacheServer) {
	ctx := context.Background()
	cs := cache.CacheServer{}
	cs.InitCache(ctx, cache.DefaultExpirationTimeFinalized, cache.DefaultExpirationForNonFinalized, cache.DisabledFlagOption)
	cacheServer := &cache.RelayerCacheServer{CacheServer: &cs}
	return ctx, cacheServer
}

func TestCacheSetGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		valid     bool
		delay     time.Duration
		finalized bool
		hash      []byte
	}{
		{name: "Finalized No Hash", valid: true, delay: time.Millisecond, finalized: true, hash: nil},
		{name: "Finalized After delay No Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: nil},
		{name: "NonFinalized No Hash", valid: true, delay: time.Millisecond, finalized: false, hash: nil},
		{name: "NonFinalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: nil},
		{name: "Finalized With Hash", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}},
		{name: "Finalized After delay With Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: []byte{1, 2, 3}},
		{name: "NonFinalized With Hash", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}},
		{name: "NonFinalized After delay With Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: []byte{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cacheServer := initTest()
			request := getRequest(1230, []byte(StubSig), StubApiInterface)

			response := &pairingtypes.RelayReply{}

			messageSet := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}

			_, err := cacheServer.SetRelay(ctx, &messageSet)
			require.Nil(t, err)

			// perform a delay
			time.Sleep(tt.delay)
			// now to get it

			messageGet := pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}
			_, err = cacheServer.GetRelay(ctx, &messageGet)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func getRequest(requestedBlock int64, data []byte, apiInterface string) *pairingtypes.RelayPrivateData {
	request := &pairingtypes.RelayPrivateData{
		ConnectionType: StubConnectionType,
		ApiUrl:         StubApiUrl,
		Data:           data,
		RequestBlock:   requestedBlock,
		ApiInterface:   apiInterface,
		Salt:           []byte{1, 2},
		Metadata:       []pairingtypes.Metadata{},
		Addon:          "",
		Extensions:     []string{},
	}
	return request
}

func shallowCopy(request *pairingtypes.RelayPrivateData) *pairingtypes.RelayPrivateData {
	return &pairingtypes.RelayPrivateData{
		ConnectionType: request.ConnectionType,
		ApiUrl:         request.ApiUrl,
		Data:           request.Data,
		RequestBlock:   request.RequestBlock,
		ApiInterface:   request.ApiInterface,
		Salt:           request.Salt,
		Metadata:       request.Metadata,
		Addon:          request.Addon,
		Extensions:     request.Extensions,
	}
}

func TestCacheGetWithoutSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		valid     bool
		delay     time.Duration
		finalized bool
		hash      []byte
	}{
		{name: "Finalized No Hash", finalized: true, hash: nil},
		{name: "Finalized After delay No Hash", finalized: true, hash: nil},
		{name: "NonFinalized No Hash", finalized: false, hash: nil},
		{name: "NonFinalized After delay No Hash", finalized: false, hash: nil},
		{name: "Finalized With Hash", finalized: true, hash: []byte{1, 2, 3}},
		{name: "Finalized After delay With Hash", finalized: true, hash: []byte{1, 2, 3}},
		{name: "NonFinalized With Hash", finalized: false, hash: []byte{1, 2, 3}},
		{name: "NonFinalized After delay With Hash", finalized: false, hash: []byte{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cacheServer := initTest()

			request := getRequest(1230, []byte(StubSig), StubApiInterface)

			// now to get it

			messageGet := pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}
			_, err := cacheServer.GetRelay(ctx, &messageGet)

			require.Error(t, err)
		})
	}
}

func TestCacheFailSetWithInvalidRequestBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		finalized           bool
		hash                []byte
		invalidRequestValue int64
	}{
		{name: "Finalized No Hash", finalized: true, hash: nil, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "Finalized After delay No Hash", finalized: true, hash: nil, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "NonFinalized No Hash", finalized: false, hash: nil, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "NonFinalized After delay No Hash", finalized: false, hash: nil, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "Finalized With Hash", finalized: true, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "Finalized After delay With Hash", finalized: true, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "NonFinalized With Hash", finalized: false, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "NonFinalized After delay With Hash", finalized: false, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.NOT_APPLICABLE},
		{name: "Finalized No Hash", finalized: true, hash: nil, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "Finalized After delay No Hash", finalized: true, hash: nil, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "NonFinalized No Hash", finalized: false, hash: nil, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "NonFinalized After delay No Hash", finalized: false, hash: nil, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "Finalized With Hash", finalized: true, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "Finalized After delay With Hash", finalized: true, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "NonFinalized With Hash", finalized: false, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.EARLIEST_BLOCK},
		{name: "NonFinalized After delay With Hash", finalized: false, hash: []byte{1, 2, 3}, invalidRequestValue: spectypes.EARLIEST_BLOCK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cacheServer := initTest()

			request := getRequest(tt.invalidRequestValue, []byte(StubSig), StubApiInterface)

			response := &pairingtypes.RelayReply{}

			messageSet := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}

			_, err := cacheServer.SetRelay(ctx, &messageSet)
			require.Error(t, err)
		})
	}
}

// all gets try to get latest, latest replacement is by definition temporary so everything with delay will fail
func TestCacheSetGetLatest(t *testing.T) {
	// all tests use the same cache server
	ctx, cacheServer := initTest()
	tests := []struct {
		name                   string
		valid                  bool
		delay                  time.Duration
		finalized              bool
		hash                   []byte
		latestBlockForSetRelay int64
		latestIsCorrect        bool
	}{
		{name: "Finalized No Hash", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1230, latestIsCorrect: true},
		{name: "Finalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1240, latestIsCorrect: true},
		{name: "NonFinalized No Hash", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1250, latestIsCorrect: true},
		{name: "NonFinalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1260, latestIsCorrect: true},
		{name: "Finalized With Hash", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: true},
		{name: "Finalized After delay With Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1280, latestIsCorrect: true},
		{name: "NonFinalized With Hash", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: true},
		{name: "NonFinalized After delay With Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1300, latestIsCorrect: true},
		{name: "Finalized No Hash update latest", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1400, latestIsCorrect: true},
		// these are now using latest that is not updated, since latest is 1400
		{name: "Finalized No Hash, with existing finalized entry", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1230, latestIsCorrect: false},
		{name: "NonFinalized No Hash, with existing finalized entry", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1250, latestIsCorrect: false},
		{name: "Finalized After delay No Hash, with expired latest entry", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1240, latestIsCorrect: false},
		{name: "NonFinalized After delay No Hash, with expired latest entry", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1260, latestIsCorrect: false},
		{name: "Finalized With Hash, with expired latest entry", valid: false, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "Finalized After delay With Hash, with expired latest entry", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1280, latestIsCorrect: false},
		{name: "NonFinalized With Hash, with expired latest entry", valid: false, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		{name: "NonFinalized After delay With Hash, with expired latest entry", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1300, latestIsCorrect: false},
		// try to set up the same latest again
		{name: "Finalized No Hash update latest", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1400, latestIsCorrect: true},
		{name: "Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		{name: "Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		// set up a new latest
		{name: "new latest, Non Finalized No Hash update latest", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1410, latestIsCorrect: true},
		{name: "new latest, Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "new latest, NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		{name: "new latest, Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "new latest, NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		// repeat now with hash
		{name: "repeat with hash, Finalized Hash update latest", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1410, latestIsCorrect: true},
		{name: "repeat with hash, Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "repeat with hash, NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		{name: "repeat with hash, Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "repeat with hash, NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		// set up a new latest now with hash
		{name: "new latest with hash, Non Finalized Hash update latest", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1420, latestIsCorrect: true},
		{name: "new latest with hash, Finalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "new latest with hash, NonFinalized With Hash, with existing latest entry", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: false},
		{name: "new latest with hash, Finalized With Hash, with existing latest entry", valid: false, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1270, latestIsCorrect: false},
		{name: "new latest with hash, NonFinalized With Hash, with existing latest entry", valid: false, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1290, latestIsCorrect: false},
	}

	var latestBlockForRelay int64 = 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if latestBlockForRelay < tt.latestBlockForSetRelay {
				latestBlockForRelay = tt.latestBlockForSetRelay
			}

			request := getRequest(tt.latestBlockForSetRelay, []byte(StubSig), StubApiInterface)

			response := &pairingtypes.RelayReply{LatestBlock: tt.latestBlockForSetRelay}
			messageSet := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}
			_ = utils.LavaFormatDebug("next test", utils.Attribute{Key: "name", Value: tt.name})
			_, err := cacheServer.SetRelay(ctx, &messageSet)
			require.Nil(t, err)
			// perform a delay
			if tt.delay > 2*time.Millisecond {
				_ = utils.LavaFormatDebug("Big Delay", utils.Attribute{Key: "delay", Value: fmt.Sprintf("%+v", tt.delay)})
			}
			time.Sleep(tt.delay)
			// now to get it

			// modify the request to get latest
			request.RequestBlock = spectypes.LATEST_BLOCK
			messageGet := pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}

			cacheReply, err := cacheServer.GetRelay(ctx, &messageGet)
			if tt.valid {
				require.Nil(t, err)
				if tt.latestIsCorrect {
					require.Equal(t, cacheReply.GetReply().LatestBlock, tt.latestBlockForSetRelay)
				} else {
					require.Greater(t, cacheReply.GetReply().LatestBlock, tt.latestBlockForSetRelay)
					require.Equal(t, cacheReply.GetReply().LatestBlock, latestBlockForRelay)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

// this test sets up a bigger latest block after the set (with a different entry) so we make sure to miss latest request
func TestCacheSetGetLatestWhenAdvancingLatest(t *testing.T) {
	// all tests use the same cache server
	ctx, cacheServer := initTest()
	tests := []struct {
		name                   string
		valid                  bool
		delay                  time.Duration
		finalized              bool
		hash                   []byte
		latestBlockForSetRelay int64
		latestIsCorrect        bool
	}{
		// latestBlockForSetRelay need to increase by more than 1 for each test
		{name: "1: Finalized No Hash", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1230, latestIsCorrect: true},
		{name: "2: Finalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1240, latestIsCorrect: true},
		{name: "3: NonFinalized No Hash", valid: true, delay: time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1250, latestIsCorrect: true},
		{name: "4: NonFinalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: nil, latestBlockForSetRelay: 1260, latestIsCorrect: true},
		{name: "5: Finalized With Hash", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1270, latestIsCorrect: true},
		{name: "6: Finalized After delay With Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1280, latestIsCorrect: true},
		{name: "7: NonFinalized With Hash", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1290, latestIsCorrect: true},
		{name: "8: NonFinalized After delay With Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: []byte{1, 2, 3}, latestBlockForSetRelay: 1300, latestIsCorrect: true},
		{name: "Non Finalized No Hash update latest", valid: true, delay: time.Millisecond, finalized: true, hash: nil, latestBlockForSetRelay: 1400, latestIsCorrect: true},
	}

	var latestBlockForRelay int64 = 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if latestBlockForRelay < tt.latestBlockForSetRelay {
				latestBlockForRelay = tt.latestBlockForSetRelay
			}

			request := getRequest(tt.latestBlockForSetRelay, []byte(StubSig), StubApiInterface)

			response := &pairingtypes.RelayReply{LatestBlock: tt.latestBlockForSetRelay}
			messageSet := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}

			_, err := cacheServer.SetRelay(ctx, &messageSet)
			require.Nil(t, err)
			// perform a delay
			time.Sleep(tt.delay)
			// now to get it

			// modify the request to get latest
			request.RequestBlock = spectypes.LATEST_BLOCK
			messageGet := pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}

			cacheReply, err := cacheServer.GetRelay(ctx, &messageGet)
			if tt.valid {
				require.Nil(t, err)
				if tt.latestIsCorrect {
					require.Equal(t, cacheReply.GetReply().LatestBlock, tt.latestBlockForSetRelay)
				} else {
					require.Greater(t, cacheReply.GetReply().LatestBlock, tt.latestBlockForSetRelay)
					require.Equal(t, cacheReply.GetReply().LatestBlock, latestBlockForRelay)
				}
			} else {
				require.Error(t, err)
			}

			request2 := shallowCopy(request)
			request2.RequestBlock = latestBlockForRelay + 1 // make latest block advance
			request2.Data = []byte(StubData + "nonRelevantData")

			messageSet2 := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request2),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}
			_, err = cacheServer.SetRelay(ctx, &messageSet2) // we use this to increase latest block received
			require.NoError(t, err)
			messageGet = pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}
			// repeat our latest block get, this time we expect it to look for a newer block and fail
			_, err = cacheServer.GetRelay(ctx, &messageGet)
			require.Error(t, err)
		})
	}
}

func TestCacheSetGetJsonRPCWithID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		valid     bool
		delay     time.Duration
		finalized bool
		hash      []byte
	}{
		{name: "Finalized No Hash", valid: true, delay: time.Millisecond, finalized: true, hash: nil},
		{name: "Finalized After delay No Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: nil},
		{name: "NonFinalized No Hash", valid: true, delay: time.Millisecond, finalized: false, hash: nil},
		{name: "NonFinalized After delay No Hash", valid: false, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: nil},
		{name: "Finalized With Hash", valid: true, delay: time.Millisecond, finalized: true, hash: []byte{1, 2, 3}},
		{name: "Finalized After delay With Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: true, hash: []byte{1, 2, 3}},
		{name: "NonFinalized With Hash", valid: true, delay: time.Millisecond, finalized: false, hash: []byte{1, 2, 3}},
		{name: "NonFinalized After delay With Hash", valid: true, delay: cache.DefaultExpirationForNonFinalized + time.Millisecond, finalized: false, hash: []byte{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cacheServer := initTest()
			id := rand.Int63()

			formatIDInJson := func(idNum int64) []byte {
				return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"status","params":[],"id":%d}`, idNum))
			}
			formatIDInJsonResponse := func(idNum int64) []byte {
				return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","result":0x12345,"id":%d}`, idNum))
			}
			request := getRequest(1230, formatIDInJson(id), spectypes.APIInterfaceJsonRPC) // &pairingtypes.RelayRequest{

			response := &pairingtypes.RelayReply{
				Data: formatIDInJsonResponse(id), // response has the old id when cached
			}

			messageSet := pairingtypes.RelayCacheSet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Response:  response,
				Finalized: tt.finalized,
			}

			_, err := cacheServer.SetRelay(ctx, &messageSet)
			require.Nil(t, err)

			// perform a delay
			time.Sleep(tt.delay)
			// now to get it

			changedID := id + 1
			// now we change the ID:
			request.Data = formatIDInJson(changedID)

			messageGet := pairingtypes.RelayCacheGet{
				Request:   shallowCopy(request),
				BlockHash: tt.hash,
				ChainID:   StubChainID,
				Finalized: tt.finalized,
			}
			cacheReply, err := cacheServer.GetRelay(ctx, &messageGet)
			if tt.valid {
				require.Nil(t, err)
				result := gjson.GetBytes(cacheReply.GetReply().Data, format.IDFieldName)
				require.Equal(t, gjson.Number, result.Type)
				extractedID := result.Int()
				require.Equal(t, changedID, extractedID)
			} else {
				require.Error(t, err)
			}
		})
	}
}
