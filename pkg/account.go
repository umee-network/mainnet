package pkg

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	umeeapp "github.com/umee-network/umee/app"
)

var (
	uumeeExponent   int64 = 1_000_000
	totalSupplyDiff int64 = 0
)

// GenerateAccount generates a genesis account given various vesting parameters.
func GenerateAccount(
	addrStr string,
	tokenAllocStr string,
	genesisTime time.Time,
	cliff int,
	vesting int,
) (authtypes.GenesisAccount, banktypes.Balance, error) {
	addr, err := ConvertAddress(addrStr)
	if err != nil {
		return nil, banktypes.Balance{}, fmt.Errorf("failed to convert address (%s): %w", addrStr, err)
	}

	tokenAlloc, err := sdk.NewDecFromStr(tokenAllocStr)
	if err != nil {
		return nil, banktypes.Balance{}, fmt.Errorf("failed to parse token allocation amount: %s", tokenAllocStr)
	}

	// convert the given token allocation in umee to the base denom uumee
	baseTokenAlloc := tokenAlloc.MulInt64(uumeeExponent).TruncateInt()

	// We require an expected total supply of 10B. However, due to truncation of
	// tiny amounts, e.g .33 uumee, the total supply might be slightly off from
	// the 10B target. So we add the remainder to one of the foundation's multisig
	// accounts, umee1pn8x9rv4823cgwmyh07z230mllg2prdxku97c7.
	if addr.String() == "umee1pn8x9rv4823cgwmyh07z230mllg2prdxku97c7" {
		// baseTokenAlloc = baseTokenAlloc.AddRaw(totalSupplyDiff)
	}

	coins := sdk.NewCoins(sdk.NewCoin(umeeapp.BondDenom, baseTokenAlloc)).Sort()
	baseAcc := authtypes.NewBaseAccount(addr, nil, 0, 0)
	balance := banktypes.Balance{
		Address: addr.String(),
		Coins:   coins,
	}

	var genAccount authtypes.GenesisAccount
	switch {
	case cliff > 0 && vesting > 0 && !genesisTime.IsZero():
		// Create a vesting account with a cliff period where the total balance
		// vests linearly over 'vesting' months.
		startTime := genesisTime.AddDate(0, cliff, 0)
		endTime := startTime.AddDate(0, vesting, 0)
		genAccount = authvesting.NewContinuousVestingAccount(baseAcc, coins, startTime.Unix(), endTime.Unix())

	case vesting > 0 && !genesisTime.IsZero():
		// Create a vesting account without a cliff where the total balance vests
		// linearly over 'vesting' months.
		endTime := genesisTime.AddDate(0, vesting, 0)
		genAccount = authvesting.NewContinuousVestingAccount(baseAcc, coins, genesisTime.Unix(), endTime.Unix())

	case cliff > 0 && vesting == 0:
		// Create a vesting account with a cliff only, i.e. the total balance is
		// vesting for the entire duration of the cliff and then immediately becomes
		// vested after the cliff is over.
		endTime := genesisTime.AddDate(0, cliff, 0)
		genAccount = authvesting.NewDelayedVestingAccount(baseAcc, coins, endTime.Unix())

	case cliff == 0 && vesting == 0:
		// create a normal non-vesting account
		genAccount = baseAcc

	default:
		err = fmt.Errorf(
			"unsupported account parameters; address: %s, cliff: %d, vesting: %d",
			addrStr, cliff, vesting,
		)
		return nil, banktypes.Balance{}, err
	}

	return genAccount, balance, nil
}
