package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		Use:   "generate-accounts [genesis-file] [accounts-path]",
		Args:  cobra.ExactArgs(2),
		Short: "Generate a mainnet genesis account",
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

			var files []string
			err = filepath.Walk(args[1], func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() {
					files = append(files, path)
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to collect account files")
			}

			// iterate over all data files containing account information
			for _, f := range files {
				fmt.Printf("Generate accounts from: %s\n", f)

				records, err := readCSV(f)
				if err != nil {
					return fmt.Errorf("failed to parse account file CSV: %w", err)
				}

				// only consider account records (ignoring meta records)
				for _, r := range collectRecords(records) {
					id := r[0]
					tokenAllocStr := sanitizeTokenAlloc(r[1])
					addrStr := r[3]
					cliffStr := r[4]
					vestingStr := r[5]

					if len(addrStr) == 0 {
						fmt.Printf("Skipping account: %s\n", id)
						continue
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

					packedGenAccounts, err := authtypes.PackAccounts(genAccounts)
					if err != nil {
						return fmt.Errorf("failed to Proto convert account: %w", err)
					}

					authGenState.Accounts = packedGenAccounts

					// update bank balances and total supply
					bankGenState.Balances = append(bankGenState.Balances, balance)
					bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)
					bankGenState.Supply = bankGenState.Supply.Add(balance.Coins...)
				}
			}

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

func collectRecords(records [][]string) [][]string {
	var addressRecords [][]string

	var idx int
	for i, r := range records {
		if strings.EqualFold(r[0], "ID Label") {
			idx = i
			break
		}
	}

	for _, r := range records[idx:] {
		if len(r[0]) == 0 || strings.EqualFold(r[0], "ID Label") {
			// skip empty or header rows
			continue
		}

		addressRecords = append(addressRecords, r)
	}

	return addressRecords
}

func sanitizeTokenAlloc(s string) string {
	return strings.Replace(s, ",", "", -1)
}
