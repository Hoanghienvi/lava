package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		specKeeper         types.SpecKeeper
		epochStorageKeeper types.EpochstorageKeeper
		projectsKeeper     types.ProjectsKeeper
		subscriptionKeeper types.SubscriptionKeeper
		planKeeper         types.PlanKeeper
		badgeTimerStore    common.TimerStore
		providerQosFS      common.FixationStore
		downtimeKeeper     types.DowntimeKeeper
		dualStakingKeeper  types.DualStakingKeeper
	}
)

// sanity checks at start time
func init() {
	if types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER == 0 {
		panic("types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0")
	}
	if types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0 {
		panic("types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0")
	}
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	epochStorageKeeper types.EpochstorageKeeper,
	projectsKeeper types.ProjectsKeeper,
	subscriptionKeeper types.SubscriptionKeeper,
	planKeeper types.PlanKeeper,
	downtimeKeeper types.DowntimeKeeper,
	dualStakingKeeper types.DualStakingKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	keeper := &Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		memKey:             memKey,
		paramstore:         ps,
		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		specKeeper:         specKeeper,
		epochStorageKeeper: epochStorageKeeper,
		projectsKeeper:     projectsKeeper,
		subscriptionKeeper: subscriptionKeeper,
		planKeeper:         planKeeper,
		downtimeKeeper:     downtimeKeeper,
		dualStakingKeeper:  dualStakingKeeper,
	}

	// note that the timer and badgeUsedCu keys are the same (so we can use only the second arg)
	badgeTimerCallback := func(ctx sdk.Context, badgeKey, _ []byte) {
		keeper.RemoveBadgeUsedCu(ctx, badgeKey)
	}
	badgeTimerStore := common.NewTimerStore(storeKey, cdc, types.BadgeTimerStorePrefix).
		WithCallbackByBlockHeight(badgeTimerCallback)
	keeper.badgeTimerStore = *badgeTimerStore

	keeper.providerQosFS = *common.NewFixationStore(storeKey, cdc, types.ProviderQosStorePrefix)

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.badgeTimerStore.Tick(ctx)

	if k.epochStorageKeeper.IsEpochStart(ctx) {
		// remove old session payments
		k.RemoveOldEpochPayment(ctx)
		// unstake any unstaking providers
		k.CheckUnstakingForCommit(ctx)
		// unstake/jail unresponsive providers
		k.UnstakeUnresponsiveProviders(ctx,
			types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER,
			types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
	}
}

func (k Keeper) InitProviderQoS(ctx sdk.Context, gs commontypes.GenesisState) {
	k.providerQosFS.Init(ctx, gs)
}

func (k Keeper) ExportProviderQoS(ctx sdk.Context) commontypes.GenesisState {
	return k.providerQosFS.Export(ctx)
}
