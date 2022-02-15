# Umee Mainnet

This repository contains the Umee genesis token distribution along with the
validator genesis transactions, and canonical genesis file.

To generate accounts from the token distribution CSV to an updated genesis file,
execute the following:

```shell
$ go run scripts/generate_accounts/main.go ./base_genesis.json data/umee_token_distribution_final.csv -o genesis.json
```

Note, the `base_genesis.json` file contains the genesis parameters only, i.e. no
accounts, balances, or gentxs.

To verify you have the correct genesis file:

```shell
$ jq -S -c -M '' genesis.json | shasum -a 256
d4264b31472a922ec05620793dd8832a3a78032b23cdaa065f38ff3803f13800  -
```
