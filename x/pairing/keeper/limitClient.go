package keeper

import (
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) GetAllowedCU(ctx sdk.Context, entry *epochstoragetypes.StakeEntry) (uint64, error) {
	var allowedCU uint64 = 0
	stakeToMaxCUMap := k.StakeToMaxCUList(ctx).List

	for _, stakeToCU := range stakeToMaxCUMap {
		if entry.Stake.IsGTE(stakeToCU.StakeThreshold) {
			allowedCU = stakeToCU.MaxComputeUnits
		} else {
			break
		}
	}
	return allowedCU, nil
}

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, relay *types.RelayRequest, clientEntry *epochstoragetypes.StakeEntry, clientAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64, providerAddr sdk.AccAddress) (ammountToPay uint64, err error) {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	if allowedCU == 0 {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
		// panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCUProvider {
		return k.LimitClientPairingsAndMarkForPenalty(ctx, clientAddr, relay.ChainID, relay.CuSum, clientEntry.Stake.Amount, relay.BlockHeight, totalCUInEpochForUserProvider, allowedCU, allowedCUProvider, providerAddr)
	}

	return relay.CuSum, nil
}

func (k Keeper) getMaxCULimitsPercentage(ctx sdk.Context) (float64, float64) {
	unpayLimitP := float64(k.UnpayLimit(ctx)) / float64(k.LimitDivisor(ctx))
	slashLimitP := float64(k.SlashLimit(ctx)) / float64(k.LimitDivisor(ctx))
	// current defaults - slashLimitP, unpayLimitP := 0.2, 0.1 // 20% , 10%
	return slashLimitP, unpayLimitP
}

func (k Keeper) GetEpochClientProviderUsedCUMap(ctx sdk.Context, clientPaymentStorage types.ClientPaymentStorage) (clientUsedCUMap types.ClientUsedCU, err error) {
	clientUsedCUMap = types.ClientUsedCU{0, make(map[string]uint64)}
	// for every unique payment of client for this epoch
	uniquePaymentStoragesClientProviderList := clientPaymentStorage.UniquePaymentStorageClientProvider
	for _, uniquePaymentStorageClientProvider := range uniquePaymentStoragesClientProviderList {
		paymentProviderAddr := k.GetProviderFromUniquePayment(ctx, *uniquePaymentStorageClientProvider)
		clientUsedCUMap.TotalUsed += uniquePaymentStorageClientProvider.UsedCU
		if _, ok := clientUsedCUMap.Providers[paymentProviderAddr]; ok {
			clientUsedCUMap.Providers[paymentProviderAddr] += uniquePaymentStorageClientProvider.UsedCU
		} else {
			clientUsedCUMap.Providers[paymentProviderAddr] = uniquePaymentStorageClientProvider.UsedCU
		}
	}
	return
}
func (k Keeper) GetAllowedCUClientEpoch(ctx sdk.Context, chainID string, epoch uint64, clientAddr sdk.AccAddress) (allowedCU uint64, err error) {
	// get current stake of client for this epoch
	currentStakeEntry, stakeErr := k.epochStorageKeeper.GetStakeEntryForClientEpoch(ctx, chainID, clientAddr, epoch)
	if stakeErr != nil {
		return 0, stakeErr
	}
	// get allowed of client for this epoch
	allowedCU, allowedCUErr := k.GetAllowedCU(ctx, currentStakeEntry)
	if allowedCUErr != nil {
		return 0, allowedCUErr
	}
	return
}

func maxU(x uint64, y uint64) uint64 {
	return uint64(math.Max(float64(x), float64(y)))
}

func (k Keeper) GetOverusedFromUsedCU(ctx sdk.Context, clientProvidersEpochUsedCUMap types.ClientUsedCU, allowedCU uint64, providerAddr sdk.AccAddress) (float64, float64, error) {
	if allowedCU <= 0 {
		return 0, 0, fmt.Errorf("lava_GetOverusedFromUsedCU was called with %d allowedCU", allowedCU)
	}
	overusedProviderPercent := float64(0.0)
	totalOverusedPercent := float64(clientProvidersEpochUsedCUMap.TotalUsed / allowedCU)
	if usedCU, exist := clientProvidersEpochUsedCUMap.Providers[providerAddr.String()]; exist {
		// TODO: ServicersToPairCount needs epoch !
		allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
		overusedCU := maxU(0, usedCU-allowedCUProvider)
		overusedProviderPercent = float64(overusedCU / allowedCUProvider)
	}
	return totalOverusedPercent, overusedProviderPercent, nil
}

func (k Keeper) GetEpochClientUsedCUMap(ctx sdk.Context, chainID string, epoch uint64, clientAddr sdk.AccAddress) (types.ClientUsedCU, error) {
	clientStoragePaymentKeyEpoch := k.GetClientPaymentStorageKey(ctx, chainID, epoch, clientAddr)
	clientPaymentStorage, found := k.GetClientPaymentStorage(ctx, clientStoragePaymentKeyEpoch)
	if found { // no payments this epoch, continue + advance epoch
		clientProvidersEpochUsedCUMap, errPaymentStorage := k.GetEpochClientProviderUsedCUMap(ctx, clientPaymentStorage)
		return clientProvidersEpochUsedCUMap, errPaymentStorage
	}
	return types.ClientUsedCU{0.0, make(map[string]uint64)}, nil
}

func (k Keeper) getOverusedCUPercentageAllEpochs(ctx sdk.Context, chainID string, clientAddr sdk.AccAddress, providerAddr sdk.AccAddress) (clientProviderOverusedPercentMap types.ClientProviderOverusedCUPercent, err error) {
	//TODO: Caching will save a lot of time...
	epochLast := k.epochStorageKeeper.GetEpochStart(ctx)
	clientProviderOverusedPercentMap = types.ClientProviderOverusedCUPercent{0.0, 0.0}

	// for every epoch in memory
	for epoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx); epoch <= epochLast; epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch) {
		// get epochPayments for this client

		clientProvidersEpochUsedCUMap, errPaymentStorage := k.GetEpochClientUsedCUMap(ctx, chainID, epoch, clientAddr)
		if errPaymentStorage != nil {
			return clientProviderOverusedPercentMap, errPaymentStorage
		} else if clientProvidersEpochUsedCUMap.TotalUsed == 0 {
			// no payments this epoch - continue
			continue
		}
		allowedCU, allowedCUErr := k.GetAllowedCUClientEpoch(ctx, chainID, epoch, clientAddr)
		if allowedCUErr != nil {
			err = allowedCUErr
			return
		} else if allowedCU == 0 {
			// user has no stake this epoch - continue
			continue
		}
		totalOverusedPercent, providerOverusedPercent, overusedErr := k.GetOverusedFromUsedCU(ctx, clientProvidersEpochUsedCUMap, allowedCU, providerAddr)
		if overusedErr != nil {
			err = overusedErr
			return
		}
		clientProviderOverusedPercentMap.TotalOverusedPercent += totalOverusedPercent
		clientProviderOverusedPercentMap.OverusedPercentProvider += providerOverusedPercent
		// clientProviderOverusedPercentMap.TotalUsed += providerOverusedPercent

		epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch)
	}
	return clientProviderOverusedPercentMap, nil
}

// func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, relay *types.RelayRequest, clientEntry *epochstoragetypes.StakeEntry, clientAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64, allowedCU uint64, allowedCUProvider uint64, providerAddr sdk.AccAddress) (amountToPay uint64, err error) {
func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, clientAddr sdk.AccAddress, chainID string, CuSum uint64, clientStakeAmount sdk.Int, BlockHeight int64, totalCUInEpochForUserProvider uint64, allowedCU uint64, allowedCUProvider uint64, providerAddr sdk.AccAddress) (CUToPay uint64, err error) {
	eventType := "lava_event"
	logger := k.Logger(ctx)
	slashLimitPercent, unpayLimitPercent := k.getMaxCULimitsPercentage(ctx)
	// clientOverusedCU, err := k.getOverusedCUPercentageAllEpochs(ctx, chainID, sdk.AccAddress(clientEntry.Address), providerAddr)
	clientOverusedCU, err := k.getOverusedCUPercentageAllEpochs(ctx, chainID, clientAddr, providerAddr)
	if err != nil {
		eventType = "lava_get_overused_cu"
		utils.LavaError(ctx, logger, eventType, map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":  strconv.FormatUint(CuSum, 10),
			"clientAddr":   clientAddr.String(),
			"providerAddr": providerAddr.String(),
			"error":        err.Error()},
			fmt.Sprintf("user %s, could not calculate overusedCU from memory: %s", clientAddr.String(), clientStakeAmount))
		return 0, fmt.Errorf("user %s, could not calculate overusedCU from memory: %s", clientAddr.String(), clientStakeAmount)
	}
	overusedSumTotalPercent := clientOverusedCU.TotalOverusedPercent
	overusedSumProviderPercent := clientOverusedCU.OverusedPercentProvider
	if overusedSumTotalPercent > slashLimitPercent || overusedSumProviderPercent > slashLimitPercent {
		k.SlashUser(ctx, clientAddr)
		eventType = "lava_slash_user"
		utils.LogLavaEvent(ctx, logger, "lava_slash_user", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":                strconv.FormatUint(CuSum, 10),
			"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
			"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
			"slashLimitPercent":          strconv.FormatFloat(slashLimitPercent, 'f', 6, 64)},
			"overuse is above the slashLimit - slashing user - not paying provider")
		return uint64(0), nil
	}
	if overusedSumTotalPercent < unpayLimitPercent && overusedSumProviderPercent < unpayLimitPercent {
		// overuse is under the limit - will allow provider to get payment
		// ? maybe needs to pay the allowedCU but not pay for overuse ?
		eventType = "lava_client_overused"
		utils.LogLavaEvent(ctx, logger, eventType, map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":                strconv.FormatUint(CuSum, 10),
			"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
			"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
			"unpayLimitPercent":          strconv.FormatFloat(unpayLimitPercent, 'f', 6, 64)},
			"overuse is under the unpayLimit - paying provider")
		return CuSum, nil
	}
	epoch := uint64(ctx.BlockHeight())
	clientUsedEpoch, usedCUerr := k.GetEpochClientUsedCUMap(ctx, chainID, epoch, clientAddr)
	if usedCUerr != nil || clientUsedEpoch.TotalUsed == 0 {
		eventType = "lava_GetEpochClientUsedCUMap"
		utils.LavaError(ctx, logger, "lava_GetEpochClientUsedCUMap", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":  strconv.FormatUint(CuSum, 10),
			"clientAddr":   clientAddr.String(),
			"providerAddr": providerAddr.String()},
			fmt.Sprintf("clientUsedEpoch.TotalUsed == %d - no payments for client %s found in blockHeight %d chainID %s",
				clientUsedEpoch.TotalUsed, clientAddr, BlockHeight, chainID))
		panic("we just added an epochPayment, so the totalUsed for this epoch can't be 0")
	}
	eventType = "lava_client_overused_unpay"
	// overused over the unpayLimit (under slashLimit) - paying provider upto the unpayLimit
	utils.LogLavaEvent(ctx, logger, eventType, map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
		"relay.CuSum":                strconv.FormatUint(CuSum, 10),
		"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
		"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
		"unpayLimitPercent":          strconv.FormatFloat(unpayLimitPercent, 'f', 6, 64)},
		"overuse is above the unpayLimit - paying provider upto the unpayLimit ")

	return uint64(math.Floor(float64(allowedCU)*1.1)) - clientUsedEpoch.TotalUsed + CuSum, nil
}

func (k Keeper) SlashUser(ctx sdk.Context, clientAddr sdk.AccAddress) {
	//TODO: jail user, and count problems
}

func (k Keeper) ClientMaxCUProvider(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry) (uint64, error) {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)

	return allowedCU, nil
}
