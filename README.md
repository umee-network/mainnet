# Umee Mainnet

This repository contains the Umee genesis token distribution along with the
validator genesis transactions, and canonical genesis file.

To generate accounts from the token distribution CSV to an updated genesis file,
execute the following:

```shell
$ go run scripts/generate_accounts/main.go ./base_genesis.json data/umee_token_distribution.csv -o genesis.json
```
