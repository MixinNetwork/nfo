# NFO

A decentralized layer to support NFT on Mixin Kernel. This MTG sends back an NFT to the receiver whenever it receives a transaction with valid mint extra.

## Run Node

Copy config.example.toml to ~/.nfo/config.toml, and fill all the app related fields.

```bash
nfo -c ~/.nfo/config.toml -d ~/.nfo/data
```
