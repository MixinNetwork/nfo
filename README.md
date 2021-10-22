# NFO

A decentralized layer to support NFT on Mixin Kernel. This MTG sends back an NFT to the receiver whenever it receives a transaction with valid mint extra.

## Mint NFT

A NFT token is uniquely minted by `group` and `id`. And you can optionally include a `hash` to make the represented content checksum permanently linked to the token.

- `group` must be a valid UUID string, which represents a collection, e.g. CryptoPunks. Everyone can start a new group if the collection not minted yet.
- `id` must be the big-endian bytes of an integer, which is the unique identifier in the group, e.g. 1234. Only the group creator can mint a new token in the group, while everyone can mint in the default group.
- `hash` is the checksum of the token represented content, e.g. image, audio, video or any media combinations.

After you have all these fields ready, you can create a memo using code below, then send 0.001XIN to the MTG with the memo attached.

```golang
nfo := nft.BuildMintNFO(group, id, hash)
memo := base64.RawURLEncoding.EncodeToString(nfo)
```

## Run Node

Copy config.example.toml to ~/.nfo/config.toml, and fill all the app related fields.

```bash
nfo -c ~/.nfo/config.toml -d ~/.nfo/data
```
