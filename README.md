# NFO

A decentralized layer to support NFT on Mixin Kernel. This MTG sends back an NFT to the receiver whenever it receives a transaction with valid mint extra.

## Mint NFT

A NFT token is uniquely minted by `collection` and `id`. And you can optionally include a `hash` to make the represented content checksum permanently linked to the token.

- `collection` must be a valid UUID string, which represents a collection, e.g. CryptoPunks. Everyone can start the collection if it is not minted yet.
- `id` must be the big-endian bytes of an integer, which is the unique identifier in the collection, e.g. 1234. Only the collection creator can mint a new token in it, while everyone can mint in the default collection.
- `hash` is the checksum of the token represented content, e.g. image, audio, video or any media combinations.

After you have all these fields ready, you can create a memo using code below, then send 0.001XIN to the MTG with the memo attached.

```golang
nfo := nft.BuildMintNFO(collection, id, hash)
memo := base64.RawURLEncoding.EncodeToString(nfo)
```

## Metadata

The MTG doesn't maintain metadata for tokens, it's up to the token creators and token browsers to generate and verify the metadata according to the token hash. We do propose a sample metadata format, and it could be easily extended for further needs.

```json
{
  "collection": {
    "id": "collection-uuid",
    "name": "collection name",
    "description": "description",
    "icon": {
      "hash": "hash of the collection icon",
      "url": "https url for the icon"
    }
  },
  "token": {
    "id": "token-identifier",
    "name": "token name",
    "description": "description",
    "icon": {
      "hash": "hash of the token icon",
      "url": "https url for the icon"
    },
    "media": {
      "hash": "hash of the token media",
      "url": "https url for the media",
      "mime": "the media mime type"
    }
  },
  "checksum": {
    "fields": ["collection.id", "collection.name", "token.id", "token.name", "token.media.hash"],
    "algorithm": "sha256"
  }
}
```

The metadata file can be modified by adding more properties, and should be verified as valid when the following code returns true.

```golang
content = concat all checksum.fields
checksum = checksum.algorithm content
if checksum not equals to hash {
  return false
}

for param in checksum.fields {
  if param is not a hash {
    continue
  }
  url = get url param for the hash
  content = fetch url
  checksum = checksum.algorithm content
  if param not equals to checksum {
    return false
  }
}
```

## Run Node

Copy config.example.toml to ~/.nfo/config.toml, and fill all the app related fields.

```bash
nfo -c ~/.nfo/config.toml -d ~/.nfo/data
```
