package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/umee-network/mainnet/pkg"
	umeeapp "github.com/umee-network/umee/app"
)

var (
	flagOutput = "output"
)

func main() {
	cmd := &cobra.Command{
		Use:   "generate-accounts [genesis-file] [token-distribution-file]",
		Args:  cobra.ExactArgs(2),
		Short: "Generate mainnet accounts from a token distribution CSV document.",
		Long: `Generate mainnet accounts from a token distribution CSV document.
		
Each account is added to the input genesis file, updating both auth and bank genesis
state. An optional output flag may be supplied to create a new genesis file instead
of updating the provided one.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			encCfg := umeeapp.MakeEncodingConfig()
			cdc := encCfg.Marshaler

			// parse genesis state from the provided file
			genesisFilePath := args[0]
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genesisFilePath)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			authGenState := authtypes.GetGenesisStateFromAppState(cdc, appState)
			genAccounts, err := authtypes.UnpackAccounts(authGenState.Accounts)
			if err != nil {
				return fmt.Errorf("failed to get accounts from genesis state: %w", err)
			}

			bankGenState := banktypes.GetGenesisStateFromAppState(cdc, appState)

			records, err := readCSV(args[1])
			if err != nil {
				return fmt.Errorf("failed to parse account file CSV: %w", err)
			}

			for i, row := range records[1:] { // skip the header row
				addrStr := row[0]
				tokenAllocStr := sanitizeTokenAlloc(row[1])
				cliffStr := row[2]
				vestingStr := row[3]

				fmt.Printf("Importing account %d (%s)\n", i, addrStr)

				// Some records have "  -   " as the cliff, which we interpret as a zero
				// cliff period.
				if strings.Contains(cliffStr, "-") {
					cliffStr = "0"
				}

				cliff, err := strconv.Atoi(cliffStr)
				if err != nil {
					return fmt.Errorf("failed to parse vesting cliff (%s): %w", cliffStr, err)
				}

				vesting, err := strconv.Atoi(vestingStr)
				if err != nil {
					return fmt.Errorf("failed to parse vesting (%s): %w", vestingStr, err)
				}

				genAcc, balance, err := pkg.GenerateAccount(addrStr, tokenAllocStr, genDoc.GenesisTime, cliff, vesting)
				if err != nil {
					return err
				}

				if genAccounts.Contains(genAcc.GetAddress()) {
					return fmt.Errorf("address already exists in genesis state: %s", genAcc.GetAddress())
				}

				// Add the new account to the set of genesis accounts and sanitize the
				// accounts afterwards.
				genAccounts = append(genAccounts, genAcc)
				genAccounts = authtypes.SanitizeGenesisAccounts(genAccounts)

				// update bank balances and total supply
				bankGenState.Balances = append(bankGenState.Balances, balance)
				bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)
				bankGenState.Supply = bankGenState.Supply.Add(balance.Coins...)
			}

			packedGenAccounts, err := authtypes.PackAccounts(genAccounts)
			if err != nil {
				return fmt.Errorf("failed to Proto convert account: %w", err)
			}

			authGenState.Accounts = packedGenAccounts

			// update auth genesis state
			bz, err := cdc.MarshalJSON(&authGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal auth genesis state: %w", err)
			}

			appState[authtypes.ModuleName] = bz

			// update bank genesis state
			bz, err = cdc.MarshalJSON(bankGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal bank genesis state: %w", err)
			}

			appState[banktypes.ModuleName] = bz

			// finally, marshal the entire application state
			bz, err = json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = bz

			// overwrite the existing or create a new genesis file
			outputPath := genesisFilePath
			if v, err := cmd.Flags().GetString(flagOutput); len(v) > 0 && err == nil {
				outputPath = v
			}

			return genutil.ExportGenesisFile(genDoc, outputPath)
		},
	}

	cmd.Flags().StringP(flagOutput, "o", "", "Write updated genesis state to file instead of overwritting existing genesis file")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readCSV(file string) ([][]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	return lines, nil
}

// sanitizeTokenAlloc removes all ',' and whitspace characters from the input
// token allocation string.
func sanitizeTokenAlloc(s string) string {
	s = strings.Replace(s, ",", "", -1)
	s = strings.TrimSpace(s)
	return s
}
