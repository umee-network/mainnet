package pkg

import (
	"fmt"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	umeeapp "github.com/umee-network/umee/app"
)

var (
	uumeeExponent = big.NewInt(1_000_000)
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

	amt := new(big.Int)
	amt, ok := amt.SetString(tokenAllocStr, 10)
	if !ok {
		return nil, banktypes.Balance{}, fmt.Errorf("failed to parse token allocation amount: %s", tokenAllocStr)
	}

	// convert the given token allocation in umee to the base denom uumee
	convertedAmt := new(big.Int).Mul(amt, uumeeExponent)

	tokenAlloc := sdk.NewIntFromBigInt(convertedAmt)
	baseAcc := authtypes.NewBaseAccount(addr, nil, 0, 0)
	coins := sdk.NewCoins(sdk.NewCoin(umeeapp.BondDenom, tokenAlloc)).Sort()
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

	case cliff == 0 && vesting == 0:
		// create a normal non-vesting account
		genAccount = baseAcc

	default:
		return nil, banktypes.Balance{}, fmt.Errorf("unsupported account parameters: TODO...")
	}

	return genAccount, balance, nil
}
