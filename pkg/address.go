package pkg

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	umeeapp "github.com/umee-network/umee/app"
)

// ConvertAddress converts a Bech32 Cosmos or Umee address to an AccAddress.
func ConvertAddress(addrStr string) (sdk.AccAddress, error) {
	var (
		bz  []byte
		err error
	)

	// Determine if we need to manually convert a cosmos Bech32 address to an umee
	// Bech32 address or if we can just parse the umee Bech32 address directly.
	if strings.HasPrefix(addrStr, sdk.Bech32PrefixAccAddr) {
		bz, err = sdk.GetFromBech32(addrStr, sdk.Bech32PrefixAccAddr)
		if err != nil {
			return nil, err
		}
	} else {
		bz, err = sdk.GetFromBech32(addrStr, umeeapp.AccountAddressPrefix)
		if err != nil {
			return nil, err
		}
	}

	if err := umeeapp.VerifyAddressFormat(bz); err != nil {
		return nil, err
	}

	return sdk.AccAddress(bz), nil
}
