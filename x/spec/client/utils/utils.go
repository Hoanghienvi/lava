package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/spec/types"
)

type (
	SpecAddProposalJSON struct {
		Proposal types.SpecAddProposal `json:"proposal"`
		Deposit  string                `json:"deposit" yaml:"deposit"`
	}
)

// Parse spec add proposal JSON form file
func ParseSpecAddProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret SpecAddProposalJSON, err error) {
	for _, fileName := range strings.Split(proposalFile, ",") {
		proposal := SpecAddProposalJSON{}

		contents, err := os.ReadFile(fileName)
		if err != nil {
			return proposal, err
		}
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.DisallowUnknownFields() // This will make the unmarshal fail if there are unused fields

		if err := decoder.Decode(&proposal); err != nil {
			return proposal, fmt.Errorf("failed in file: %s, error %w", fileName, err)
		}
		// if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		// 	return proposal, err
		// }
		if len(ret.Proposal.Specs) > 0 {
			for _, spec := range proposal.Proposal.Specs {
				if spec.Name == "" {
					utils.LavaFormatFatal("invalid spec name for spec", nil,
						utils.Attribute{Key: "spec", Value: spec},
						utils.Attribute{Key: "filename", Value: fileName},
						utils.Attribute{Key: "other specs", Value: proposal.Proposal.Specs},
					)
				}
			}
			ret.Proposal.Specs = append(ret.Proposal.Specs, proposal.Proposal.Specs...)
			ret.Proposal.Description = proposal.Proposal.Description + " " + ret.Proposal.Description
			ret.Proposal.Title = "Multi_Spec_Add"
			retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
			if err != nil {
				return proposal, err
			}
			proposalDeposit, err := sdk.ParseCoinNormalized(proposal.Deposit)
			if err != nil {
				return proposal, err
			}
			ret.Deposit = retDeposit.Add(proposalDeposit).String()
		} else {
			ret = proposal
		}
	}
	return ret, nil
}
