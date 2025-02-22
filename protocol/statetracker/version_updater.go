package statetracker

import (
	"context"
	"sync"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	CallbackKeyForVersionUpdate = "version-update"
)

type VersionStateQuery interface {
	GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error)
}

type VersionValidationInf interface {
	ValidateProtocolVersion(lastKnownVersion *protocoltypes.Version) error
}

type VersionUpdater struct {
	lock                 sync.RWMutex
	eventTracker         *EventTracker
	versionStateQuery    VersionStateQuery
	lastKnownVersion     *protocoltypes.Version
	VersionValidationInf // embedding the interface, this tells: VersionUpdater has ValidateProtocolVersion method
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker, version *protocoltypes.Version, versionValidator VersionValidationInf) *VersionUpdater {
	return &VersionUpdater{versionStateQuery: versionStateQuery, eventTracker: eventTracker, lastKnownVersion: version, VersionValidationInf: versionValidator}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

func (vu *VersionUpdater) RegisterVersionUpdatable() {
	vu.lock.RLock()
	defer vu.lock.RUnlock()
	err := vu.ValidateProtocolVersion(vu.lastKnownVersion)
	if err != nil {
		utils.LavaFormatError("Protocol Version Error", err)
	}
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	versionUpdated := vu.eventTracker.getLatestVersionEvents()
	if versionUpdated {
		// fetch updated version from consensus
		version, err := vu.versionStateQuery.GetProtocolVersion(context.Background())
		if err != nil {
			utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
			return
		}
		utils.LavaFormatInfo("Protocol version has been fetched successfully!",
			utils.Attribute{Key: "old_version", Value: vu.lastKnownVersion},
			utils.Attribute{Key: "new_version", Value: version})
		// if no error, set the last known version.
		vu.lastKnownVersion = version
	}
	// monitor protocol version on each new block
	err := vu.ValidateProtocolVersion(vu.lastKnownVersion)
	if err != nil {
		utils.LavaFormatError("Validate Protocol Version Error", err)
	}
}
